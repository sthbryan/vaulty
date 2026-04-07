package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"github.com/sthbryan/vaulty/internal/config"
	"github.com/sthbryan/vaulty/internal/crypto"
	"github.com/sthbryan/vaulty/internal/storage"
	"github.com/sthbryan/vaulty/internal/ui"
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

func findMetadataUser(metadata *config.Metadata, username string) (*config.UserEntry, error) {
	for i := range metadata.Users {
		if metadata.Users[i].Username == username {
			return &metadata.Users[i], nil
		}
	}
	return nil, fmt.Errorf("user %q not found", username)
}

func prepareTransferMetadata(metadata *config.Metadata, currentOwner, newOwner string) (*config.Metadata, error) {
	if metadata.Owner != currentOwner {
		return nil, fmt.Errorf("current config owner mismatch: metadata owner is %q", metadata.Owner)
	}

	if _, err := findMetadataUser(metadata, newOwner); err != nil {
		return nil, fmt.Errorf("new owner not found: %s", newOwner)
	}

	metadata.Owner = newOwner
	for i := range metadata.Users {
		if metadata.Users[i].Username == currentOwner {
			metadata.Users[i].Role = "editor"
		}
		if metadata.Users[i].Username == newOwner {
			metadata.Users[i].Role = "owner"
		}
	}

	return metadata, nil
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

	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
	defer cancel()

	factory := storage.NewFactory(cfg)
	s, err := factory.CreateStorage()
	if err != nil {
		return fmt.Errorf("creating storage: %w", err)
	}

	metadataBytes, err := s.GetMetadata(ctx)
	if err != nil {
		return fmt.Errorf("failed to get metadata: %w", err)
	}

	var metadata config.Metadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return fmt.Errorf("failed to parse metadata: %w", err)
	}

	ownerEntry, err := findMetadataUser(&metadata, cfg.CurrentUser)
	if err != nil {
		return fmt.Errorf("current user not found in metadata: %w", err)
	}

	if _, err := findMetadataUser(&metadata, newOwner); err != nil {
		return fmt.Errorf("new owner not found: %s", newOwner)
	}

	fmt.Println()
	ui.PrintInfo("Transfer ownership to %s?", newOwner)
	ui.PrintInfo("You will become an editor.")
	fmt.Println()

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

	if ownerEntry.PasswordChallenge == nil {
		return fmt.Errorf("current owner does not have a password challenge set up")
	}

	isValid := crypto.ValidatePasswordWithChallenge(
		password,
		ownerEntry.PasswordChallenge.Salt,
		ownerEntry.PasswordChallenge.Challenge,
	)
	if !isValid {
		return fmt.Errorf("invalid password")
	}

	updatedMetadata, err := prepareTransferMetadata(&metadata, cfg.CurrentUser, newOwner)
	if err != nil {
		return err
	}

	ui.PrintInfo("Uploading metadata...")
	updatedMetadataBytes, err := json.MarshalIndent(updatedMetadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if err := s.PutMetadata(ctx, updatedMetadataBytes); err != nil {
		return fmt.Errorf("failed to upload metadata: %w", err)
	}

	cfg.CurrentUserRole = "editor"
	cfg.Metadata = updatedMetadata

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
