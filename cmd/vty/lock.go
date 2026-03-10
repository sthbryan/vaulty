package main

import (
	"context"
	"fmt"
	"time"

	"github.com/DeadBryam/vaulty/internal/config"
	"github.com/DeadBryam/vaulty/internal/crypto"
	"github.com/DeadBryam/vaulty/internal/github"
	"github.com/DeadBryam/vaulty/internal/session"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

var lockCmd = &cobra.Command{
	Use:   "lock",
	Short: "Lock the vault session",
	Long: `Lock the vault to clear the master key from memory.

The vault data remains cached with its TTL intact, and the current user
configuration is preserved. Run 'vty unlock' to restore access.`,
	RunE: runLock,
}

func runLock(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if cfg.CurrentUser == "" {
		return fmt.Errorf("no active session - run 'vty init' or 'vty recover' first")
	}

	sm := session.GetManager()
	sess := sm.Get(cfg.CurrentUser)
	if sess == nil || !sess.IsActive() {
		return fmt.Errorf("session not active")
	}

	var password string
	err = huh.NewInput().
		Title("Confirm password").
		Placeholder("Enter your master password").
		EchoMode(huh.EchoModePassword).
		Value(&password).
		Run()
	if err != nil {
		return fmt.Errorf("form cancelled")
	}

	token, err := github.GetGitHubToken()
	if err != nil {
		return fmt.Errorf("GitHub authentication: %w", err)
	}

	client := github.NewClient(token)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	owner, repo, err := github.ParseRepo(cfg.Repo)
	if err != nil {
		return fmt.Errorf("parsing repo: %w", err)
	}

	keyPath := fmt.Sprintf("keys/%s.enc", cfg.CurrentUser)
	content, err := client.GetContent(ctx, owner, repo, keyPath)
	if err != nil {
		return fmt.Errorf("downloading keys: %w", err)
	}

	encryptedKeyData, err := client.DecodeContent(content)
	if err != nil {
		return fmt.Errorf("decoding key: %w", err)
	}

	encryptedData, err := crypto.DeserializeEncryptedData(encryptedKeyData)
	if err != nil {
		return fmt.Errorf("deserializing key: %w", err)
	}

	_, err = crypto.DecryptMasterKeyWithPassword(encryptedData, password)
	if err != nil {
		return fmt.Errorf("wrong password")
	}

	sess.Lock()

	fmt.Println("🔒 Locked - run 'vty unlock' to access vault")

	return nil
}

func init() {
	rootCmd.AddCommand(lockCmd)
}
