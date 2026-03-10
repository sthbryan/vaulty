package vault

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/DeadBryam/vaulty/internal/cache"
	"github.com/DeadBryam/vaulty/internal/config"
	"github.com/DeadBryam/vaulty/internal/crypto"
	"github.com/DeadBryam/vaulty/internal/github"
	"github.com/DeadBryam/vaulty/internal/password"
	"github.com/DeadBryam/vaulty/internal/session"
)

func ValidateAndLoadVault(ctx context.Context, cfg *config.Config, ghClient *github.Client, owner, repo string) ([]byte, error) {
	sessionMgr := session.GetManager()
	sess := sessionMgr.Get(cfg.CurrentUser)

	if sess != nil {
		return sess.VaultData, nil
	}

	passStorage, err := password.NewStorage()
	if err != nil {
		return nil, fmt.Errorf("password storage: %w", err)
	}

	cacheManager := cache.NewCacheManager(passStorage)
	if valid, _ := cacheManager.IsValid(cfg.CurrentUser); valid {
		vaultData, err := cacheManager.Load(cfg.CurrentUser)
		if err == nil {
			sess := &session.Session{
				Username:  cfg.CurrentUser,
				Role:      cfg.CurrentUserRole,
				VaultData: vaultData,
			}
			sessionMgr.Create(sess)
			return vaultData, nil
		}
	}

	metadata, err := ghClient.GetMetadata(ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata: %w", err)
	}

	var meta config.Metadata
	if err := json.Unmarshal(metadata, &meta); err != nil {
		return nil, fmt.Errorf("invalid metadata: %w", err)
	}

	userExists := false
	for _, u := range meta.Users {
		if u.Username == cfg.CurrentUser {
			userExists = true
			break
		}
	}

	if !userExists {
		return nil, fmt.Errorf("user %s not found in vault - run 'vty unlink'", cfg.CurrentUser)
	}

	pwd, err := passStorage.Get()
	if err != nil || pwd == "" {
		return nil, fmt.Errorf("password not found - run 'vty unlock'")
	}

	userKeysResp, err := ghClient.GetUserKeys(ctx, owner, repo, cfg.CurrentUser)
	if err != nil {
		return nil, fmt.Errorf("failed to get user keys: %w", err)
	}

	userKeysData, err := ghClient.DecodeContent(userKeysResp)
	if err != nil {
		return nil, fmt.Errorf("failed to decode user keys: %w", err)
	}

	masterKey, err := crypto.DecryptMasterKeyWithPassword(
		&crypto.EncryptedData{
			Salt:       userKeysData[:32],
			IV:         userKeysData[32:44],
			Ciphertext: userKeysData[44:],
		},
		pwd,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt masterKey: %w", err)
	}

	vaultResp, err := ghClient.GetContent(ctx, owner, repo, ".vaulty/vault.enc")
	if err != nil {
		return nil, fmt.Errorf("failed to get vault: %w", err)
	}

	vaultEncoded, err := ghClient.DecodeContent(vaultResp)
	if err != nil {
		return nil, fmt.Errorf("failed to decode vault: %w", err)
	}

	vaultData, err := crypto.DecryptVaultData(
		&crypto.EncryptedData{
			Salt:       vaultEncoded[:32],
			IV:         vaultEncoded[32:44],
			Ciphertext: vaultEncoded[44:],
		},
		masterKey,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt vault: %w", err)
	}

	if err := cacheManager.Save(cfg.CurrentUser, vaultData); err != nil {
		return nil, fmt.Errorf("failed to cache vault: %w", err)
	}

	sess = &session.Session{
		Username:  cfg.CurrentUser,
		Role:      cfg.CurrentUserRole,
		MasterKey: masterKey,
		VaultData: vaultData,
	}
	sessionMgr.Create(sess)

	return vaultData, nil
}
