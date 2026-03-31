package auth

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/DeadBryam/vaulty/internal/cache"
	"github.com/DeadBryam/vaulty/internal/config"
	"github.com/DeadBryam/vaulty/internal/crypto"
	"github.com/DeadBryam/vaulty/internal/password"
	"github.com/DeadBryam/vaulty/internal/session"
	"github.com/DeadBryam/vaulty/internal/storage"
)

type LoginUseCase struct {
	storageFactory *storage.Factory
}

func NewLoginUseCase(factory *storage.Factory) *LoginUseCase {
	return &LoginUseCase{storageFactory: factory}
}

type LoginInput struct {
	Username      string
	MasterPassword string
}

type LoginOutput struct {
	Session *session.Session
	Metadata *config.Metadata
}

func (uc *LoginUseCase) Execute(ctx context.Context, input LoginInput) (*LoginOutput, error) {
	s, err := uc.storageFactory.CreateStorage()
	if err != nil {
		return nil, fmt.Errorf("creating storage: %w", err)
	}

	metadataBytes, err := s.GetMetadata(ctx)
	if err != nil {
		return nil, fmt.Errorf("downloading metadata: %w", err)
	}

	var metadata config.Metadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return nil, fmt.Errorf("parsing metadata: %w", err)
	}

	var userEntry *config.UserEntry
	for i := range metadata.Users {
		if metadata.Users[i].Username == input.Username {
			userEntry = &metadata.Users[i]
			break
		}
	}

	if userEntry == nil {
		return nil, fmt.Errorf("user %q not found in vault metadata", input.Username)
	}

	if userEntry.PasswordChallenge != nil {
		if !crypto.ValidatePasswordWithChallenge(input.MasterPassword, userEntry.PasswordChallenge.Salt, userEntry.PasswordChallenge.Challenge) {
			return nil, fmt.Errorf("password validation failed")
		}
	}

	keyData, err := s.GetUserKeys(ctx, input.Username)
	if err != nil {
		return nil, fmt.Errorf("downloading user keys: %w", err)
	}

	keyJSON, err := crypto.DecompressHex(string(keyData))
	if err != nil {
		return nil, fmt.Errorf("decompression failed: %w", err)
	}

	encryptedKey := &crypto.EncryptedData{}
	if err := json.Unmarshal(keyJSON, encryptedKey); err != nil {
		return nil, fmt.Errorf("parsing master key JSON: %w", err)
	}

	masterKey, err := crypto.DecryptMasterKeyWithPassword(encryptedKey, input.MasterPassword)
	if err != nil {
		return nil, fmt.Errorf("decryption failed")
	}

	vaultDataBytes, err := s.GetVault(ctx)
	if err != nil {
		return nil, fmt.Errorf("downloading vault: %w", err)
	}

	vaultJSON, err := crypto.DecompressHex(string(vaultDataBytes))
	if err != nil {
		return nil, fmt.Errorf("vault decompression failed: %w", err)
	}

	encryptedVault := &crypto.EncryptedData{}
	if err := json.Unmarshal(vaultJSON, encryptedVault); err != nil {
		return nil, fmt.Errorf("parsing vault JSON: %w", err)
	}

	vaultData, err := crypto.DecryptVaultData(encryptedVault, masterKey)
	if err != nil {
		return nil, fmt.Errorf("vault decryption failed")
	}

	sess := session.NewSession(input.Username, userEntry.Role, masterKey, vaultData)

	return &LoginOutput{
		Session:  sess,
		Metadata: &metadata,
	}, nil
}

func (uc *LoginUseCase) SaveSession(sess *session.Session, cfg *config.Config, metadata *config.Metadata, masterPassword string) error {
	session.GetManager().Create(sess)

	cfg.SetCurrentUser(sess.Username, sess.Role)
	if cfg.Metadata == nil {
		cfg.Metadata = metadata
	} else {
		cfg.Metadata.Users = metadata.Users
	}

	if err := cfg.Save(""); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	passStorage, err := password.NewStorage()
	if err != nil {
		return fmt.Errorf("password storage: %w", err)
	}

	if err := passStorage.Set(masterPassword); err != nil {
		return fmt.Errorf("storing password: %w", err)
	}

	cacheManager := cache.NewCacheManager(passStorage)
	if err := cacheManager.Save(sess.Username, sess.VaultData); err != nil {
		return fmt.Errorf("caching vault: %w", err)
	}

	return nil
}
