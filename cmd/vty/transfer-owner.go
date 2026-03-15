package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/DeadBryam/vaulty/internal/config"
	"github.com/DeadBryam/vaulty/internal/crypto"
	"github.com/DeadBryam/vaulty/internal/github"
	"github.com/DeadBryam/vaulty/internal/ui"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

var transferOwnerCmd = &cobra.Command{
	Use:   "transfer-owner <newowner>",
	Short: "Transfer ownership to another user",
	Long: `Transfer ownership to another user in the vault.

You must be the current owner to transfer ownership.
The new owner must exist in the vault metadata.

Examples:
  vty transfer-owner juan`,
	Args: cobra.ExactArgs(1),
	RunE: runTransferOwner,
}

func runTransferOwner(cmd *cobra.Command, args []string) error {
	newOwner := args[0]

	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if cfg.IsLocalMode() {
		return fmt.Errorf("user management is not supported in local mode. Local vaults are single-owner only")
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	if !cfg.IsOwner() {
		return fmt.Errorf("only the owner can transfer ownership")
	}

	userEntry, err := cfg.FindUser(newOwner)
	if err != nil {
		return fmt.Errorf("new owner not found: %s", newOwner)
	}

	fmt.Println()
	ui.PrintInfo("Transfer ownership to %s?", newOwner)
	ui.PrintInfo("You will become an editor.")
	fmt.Println()

	token, err := github.GetGitHubToken()
	if err != nil {
		return fmt.Errorf("failed to get GitHub token: %w", err)
	}

	client := github.NewClient(token)
	ctx := context.Background()

	owner, repoName, err := github.ParseRepo(cfg.Repo)
	if err != nil {
		return fmt.Errorf("invalid repo format: %w", err)
	}

	confirmed, err := ui.AskConfirm(fmt.Sprintf("Transfer ownership to %s?", newOwner), false)
	if err != nil {
		return fmt.Errorf("confirmation failed: %w", err)
	}
	if !confirmed {
		fmt.Println()
		ui.PrintInfo("Transfer cancelled")
		return nil
	}

	fmt.Println()
	var password string
	err = huh.NewInput().
		Title("Enter your password to verify").
		Placeholder("Your master password").
		EchoMode(huh.EchoModePassword).
		Value(&password).
		Validate(func(s string) error {
			if s == "" {
				return fmt.Errorf("password cannot be empty")
			}
			return nil
		}).
		Run()
	if err != nil {
		return fmt.Errorf("form cancelled")
	}

	if userEntry.PasswordChallenge == nil {
		return fmt.Errorf("user %s does not have a password challenge set up", newOwner)
	}

	isValid := crypto.ValidatePasswordWithChallenge(
		password,
		userEntry.PasswordChallenge.Salt,
		userEntry.PasswordChallenge.Challenge,
	)
	if !isValid {
		return fmt.Errorf("invalid password")
	}

	ui.PrintInfo("Downloading metadata...")
	metadataBytes, err := client.GetMetadata(ctx, owner, repoName)
	if err != nil {
		return fmt.Errorf("failed to get metadata: %w", err)
	}

	var metadata config.Metadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return fmt.Errorf("failed to parse metadata: %w", err)
	}

	metadata.Owner = newOwner

	for i := range metadata.Users {
		if metadata.Users[i].Username == cfg.CurrentUser {
			metadata.Users[i].Role = "editor"
		}
		if metadata.Users[i].Username == newOwner {
			metadata.Users[i].Role = "owner"
		}
	}

	ui.PrintInfo("Uploading metadata...")
	updatedMetadataBytes, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if err := client.PutMetadata(ctx, owner, repoName, updatedMetadataBytes); err != nil {
		return fmt.Errorf("failed to upload metadata: %w", err)
	}

	cfg.CurrentUserRole = "editor"
	cfg.Metadata = &metadata

	if err := cfg.Save(""); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println()
	ui.PrintSuccess("Ownership transferred to %s", newOwner)
	fmt.Println()

	return nil
}

func init() {
	rootCmd.AddCommand(transferOwnerCmd)
}
