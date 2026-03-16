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
	"github.com/DeadBryam/vaulty/internal/storage"
)

func ensureAuthenticated(cfg *config.Config) (*session.Session, error) {
	if cfg.Repo == "" && !cfg.IsLocalMode() {
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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var s storage.Storage
	if cfg.IsLocalMode() {
		localStorage, err := storage.NewLocalStorage()
		if err != nil {
			return nil, fmt.Errorf("initializing local storage: %w", err)
		}
		s = localStorage
	} else {
		token, err := github.GetGitHubToken()
		if err != nil {
			return nil, fmt.Errorf("github authentication: %w", err)
		}
		var err2 error
		s, err2 = storage.NewGitHubStorage(token, cfg.Repo)
		if err2 != nil {
			return nil, fmt.Errorf("invalid repository format: %w", err2)
		}
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

	keyData, err := s.GetUserKeys(ctx, cfg.CurrentUser)
	if err != nil {
		return nil, fmt.Errorf("downloading user keys: %w", err)
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

	vaultDataBytes, err := s.GetVault(ctx)
	if err != nil {
		return nil, fmt.Errorf("downloading vault: %w", err)
	}

	vaultJSON, err := crypto.DecompressHex(string(vaultDataBytes))
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
