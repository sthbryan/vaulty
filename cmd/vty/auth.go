package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/DeadBryam/vaulty/internal/config"
	"github.com/DeadBryam/vaulty/internal/crypto"
	"github.com/DeadBryam/vaulty/internal/github"
	"github.com/DeadBryam/vaulty/internal/password"
	"github.com/DeadBryam/vaulty/internal/session"
)

func ensureAuthenticated(cfg *config.Config) (*session.Session, error) {
	if cfg.Repo == "" {
		return nil, fmt.Errorf("vaulty not initialized. Run 'vty init' first")
	}

	if cfg.CurrentUser == "" {
		return nil, fmt.Errorf("no current user set. Run 'vty login' first")
	}

	sessionManager := session.GetManager()
	existingSession := sessionManager.Get(cfg.CurrentUser)
	if existingSession != nil && existingSession.IsActive() {
		return existingSession, nil
	}

	passStorage, err := password.NewStorage()
	if err != nil {
		return nil, fmt.Errorf("password storage: %w", err)
	}

	storedPassword, err := passStorage.Get()
	if err != nil {
		return nil, fmt.Errorf("no active session. Run 'vty login' first")
	}

	sess, err := authenticateUser(cfg, storedPassword)
	if err != nil {
		return nil, err
	}

	sessionManager.Create(sess)

	return sess, nil
}

func authenticateUser(cfg *config.Config, password string) (*session.Session, error) {
	token, err := github.GetGitHubToken()
	if err != nil {
		return nil, fmt.Errorf("github authentication: %w", err)
	}

	client := github.NewClient(token)
	owner, repo, err := github.ParseRepo(cfg.Repo)
	if err != nil {
		return nil, fmt.Errorf("invalid repository format: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	metadataResp, err := client.GetContent(ctx, owner, repo, ".vaulty/metadata.vty")
	if err != nil {
		return nil, fmt.Errorf("downloading metadata: %w", err)
	}

	metadataEncData, err := client.DecodeContent(metadataResp)
	if err != nil {
		return nil, fmt.Errorf("decoding metadata: %w", err)
	}

	var metadata config.Metadata
	if err := json.Unmarshal(metadataEncData, &metadata); err != nil {
		return nil, fmt.Errorf("parsing metadata: %w", err)
	}

	var userEntry *config.UserEntry
	for i := range metadata.Users {
		if metadata.Users[i].Username == cfg.CurrentUser {
			userEntry = &metadata.Users[i]
			break
		}
	}

	if userEntry == nil {
		return nil, fmt.Errorf("user %q not found in vault metadata", cfg.CurrentUser)
	}

	if userEntry.PasswordChallenge != nil {
		if !crypto.ValidatePasswordWithChallenge(password, userEntry.PasswordChallenge.Salt, userEntry.PasswordChallenge.Challenge) {
			return nil, fmt.Errorf("password validation failed")
		}
	}

	keyPath := fmt.Sprintf(".vaulty/keys/%s.vty", cfg.CurrentUser)
	keyResp, err := client.GetContent(ctx, owner, repo, keyPath)
	if err != nil {
		return nil, fmt.Errorf("downloading user keys: %w", err)
	}

	keyData, err := client.DecodeContent(keyResp)
	if err != nil {
		return nil, fmt.Errorf("decoding key data: %w", err)
	}

	keyJSON, err := crypto.DecompressHex(string(keyData))
	if err != nil {
		return nil, fmt.Errorf("decompressing master key: %w", err)
	}

	encryptedKey := &crypto.EncryptedData{}
	if err := json.Unmarshal(keyJSON, encryptedKey); err != nil {
		return nil, fmt.Errorf("parsing master key JSON: %w", err)
	}

	masterKey, err := crypto.DecryptMasterKeyWithPassword(encryptedKey, password)
	if err != nil {
		return nil, fmt.Errorf("decrypting master key: %w", err)
	}

	vaultResp, err := client.GetContent(ctx, owner, repo, ".vaulty/vault.vty")
	if err != nil {
		return nil, fmt.Errorf("downloading vault: %w", err)
	}

	vaultEncData, err := client.DecodeContent(vaultResp)
	if err != nil {
		return nil, fmt.Errorf("decoding vault data: %w", err)
	}

	vaultJSON, err := crypto.DecompressHex(string(vaultEncData))
	if err != nil {
		return nil, fmt.Errorf("decompressing vault: %w", err)
	}

	encryptedVault := &crypto.EncryptedData{}
	if err := json.Unmarshal(vaultJSON, encryptedVault); err != nil {
		return nil, fmt.Errorf("parsing vault JSON: %w", err)
	}

	vaultData, err := crypto.DecryptVaultData(encryptedVault, masterKey)
	if err != nil {
		return nil, fmt.Errorf("vault decryption failed: %w", err)
	}

	sess := session.NewSession(cfg.CurrentUser, userEntry.Role, masterKey, vaultData)

	return sess, nil
}
