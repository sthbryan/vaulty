package users

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/DeadBryam/vaulty/internal/config"
	"github.com/DeadBryam/vaulty/internal/crypto"
	"github.com/DeadBryam/vaulty/internal/storage"
)

type AddUserInput struct {
	Username      string
	Role          string
	OwnerPassword string
}

type AddUserOutput struct {
	RecoverySeed string
}

type AddUserUseCase struct {
	storageFactory *storage.Factory
}

func NewAddUserUseCase(factory *storage.Factory) *AddUserUseCase {
	return &AddUserUseCase{storageFactory: factory}
}

func ownerUsernameForKeyLookup(metadataOwner, storageOwner string) string {
	if metadataOwner != "" {
		return metadataOwner
	}
	return storageOwner
}

func (uc *AddUserUseCase) Execute(ctx context.Context, input AddUserInput) (*AddUserOutput, error) {
	s, err := uc.storageFactory.CreateStorage()
	if err != nil {
		return nil, fmt.Errorf("creating storage: %w", err)
	}

	metadataBytes, err := s.GetMetadata(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to download metadata: %w", err)
	}

	var metadata config.Metadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return nil, fmt.Errorf("parsing metadata: %w", err)
	}

	var ownerEntry *config.UserEntry
	for i := range metadata.Users {
		if metadata.Users[i].Username == metadata.Owner {
			ownerEntry = &metadata.Users[i]
			break
		}
	}

	if ownerEntry == nil {
		return nil, fmt.Errorf("owner entry not found in metadata")
	}

	if ownerEntry.PasswordChallenge != nil {
		if !crypto.ValidatePasswordWithChallenge(input.OwnerPassword, ownerEntry.PasswordChallenge.Salt, ownerEntry.PasswordChallenge.Challenge) {
			return nil, fmt.Errorf("password validation failed")
		}
	}

	owner := ownerUsernameForKeyLookup(metadata.Owner, s.GetOwner())

	keyData, err := s.GetUserKeys(ctx, owner)
	if err != nil {
		return nil, fmt.Errorf("failed to download owner key: %w", err)
	}

	keyJSON, err := crypto.DecompressHex(string(keyData))
	if err != nil {
		return nil, fmt.Errorf("decompressing owner key: %w", err)
	}

	encryptedData := &crypto.EncryptedData{}
	if err := json.Unmarshal(keyJSON, encryptedData); err != nil {
		return nil, fmt.Errorf("parsing owner key JSON: %w", err)
	}

	masterKey, err := crypto.DecryptMasterKeyWithPassword(encryptedData, input.OwnerPassword)
	if err != nil {
		return nil, fmt.Errorf("decryption failed")
	}

	for _, user := range metadata.Users {
		if user.Username == input.Username {
			return nil, fmt.Errorf("user %q already exists", input.Username)
		}
	}

	encryptedMasterKey, err := crypto.EncryptMasterKeyWithPassword(masterKey, input.OwnerPassword)
	if err != nil {
		return nil, fmt.Errorf("encrypting master key: %w", err)
	}

	salt, challenge, err := crypto.GeneratePasswordChallengeStruct(input.OwnerPassword)
	if err != nil {
		return nil, fmt.Errorf("generating password challenge: %w", err)
	}

	newUserChallenge := &config.PasswordChallenge{
		Salt:      salt,
		Challenge: challenge,
	}

	masterKeyJSON, err := json.Marshal(encryptedMasterKey)
	if err != nil {
		return nil, fmt.Errorf("marshaling master key: %w", err)
	}

	masterKeyHex, err := crypto.CompressHex(masterKeyJSON)
	if err != nil {
		return nil, fmt.Errorf("compressing master key: %w", err)
	}

	recoverySeeds, err := crypto.GenerateRecoverySeed()
	if err != nil {
		return nil, fmt.Errorf("generating recovery seed: %w", err)
	}

	metadata.Users = append(metadata.Users, config.UserEntry{
		Username:          input.Username,
		Role:              input.Role,
		CreatedAt:         time.Now(),
		PasswordChallenge: newUserChallenge,
	})

	metadataJSON, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling metadata: %w", err)
	}

	err = s.PutUserKeys(ctx, input.Username, []byte(masterKeyHex))
	if err != nil {
		return nil, fmt.Errorf("uploading key: %w", err)
	}

	encryptedSeed, err := crypto.EncryptRecoverySeed(recoverySeeds, input.OwnerPassword)
	if err != nil {
		return nil, fmt.Errorf("encrypting recovery seed: %w", err)
	}

	encryptedSeedJSON, err := json.Marshal(encryptedSeed)
	if err != nil {
		return nil, fmt.Errorf("marshaling encrypted seed: %w", err)
	}

	recoveryHex, err := crypto.CompressHex(encryptedSeedJSON)
	if err != nil {
		return nil, fmt.Errorf("compressing recovery: %w", err)
	}

	err = s.PutRecoverySeed(ctx, input.Username, []byte(recoveryHex))
	if err != nil {
		return nil, fmt.Errorf("uploading recovery seed: %w", err)
	}

	err = s.PutMetadata(ctx, metadataJSON)
	if err != nil {
		return nil, fmt.Errorf("uploading metadata: %w", err)
	}

	return &AddUserOutput{
		RecoverySeed: recoverySeeds,
	}, nil
}
