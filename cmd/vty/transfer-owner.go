package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/DeadBryam/vaulty/internal/config"
	"github.com/DeadBryam/vaulty/internal/crypto"
	"github.com/DeadBryam/vaulty/internal/github"
	"github.com/DeadBryam/vaulty/internal/ui"
	"github.com/spf13/cobra"
)

var transferOwnerCmd = &cobra.Command{
	Use:   "transfer-owner <newowner>",
	Short: "Transfer ownership to another user",
	Long: `Transfer ownership to another user in the vault.

You must be the current owner to transfer ownership.
The new owner must exist in the metadata.`,
	Args: cobra.ExactArgs(1),
	RunE: runTransferOwner,
}

func runTransferOwner(cmd *cobra.Command, args []string) error {
	newOwner := args[0]

	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	// Validate: current user is owner
	if !cfg.IsOwner() {
		return fmt.Errorf("only the owner can transfer ownership")
	}

	// Validate: newowner exists in metadata.json
	if _, err := cfg.FindUser(newOwner); err != nil {
		return fmt.Errorf("new owner not found: %s", newOwner)
	}

	fmt.Println()
	ui.PrintInfo("Transfer ownership to %s?", newOwner)
	ui.PrintInfo("You will become an editor.")
	fmt.Println()

	// Download metadata.json
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

	// Prompt confirmation
	confirmed, err := ui.AskConfirm(fmt.Sprintf("Transfer ownership to %s?", newOwner), false)
	if err != nil {
		return fmt.Errorf("confirmation failed: %w", err)
	}
	if !confirmed {
		fmt.Println()
		ui.PrintInfo("Transfer cancelled")
		return nil
	}

	// Prompt password to verify
	fmt.Println()
	password, err := ui.AskPassword("Enter your password to verify")
	if err != nil {
		return fmt.Errorf("failed to get password: %w", err)
	}

	// Verify password with canary
	canaryResp, err := client.GetContent(ctx, owner, repoName, ".vaulty/canary.vty")
	if err != nil {
		return fmt.Errorf("failed to get canary: %w", err)
	}

	canaryData, err := client.DecodeContent(canaryResp)
	if err != nil {
		return fmt.Errorf("failed to decode canary: %w", err)
	}

	if err := crypto.ValidateCanary(canaryData, password, cfg.DeviceSalt); err != nil {
		return fmt.Errorf("invalid password")
	}

	ui.PrintInfo("Downloading metadata...")
	metadataBytes, err := client.GetMetadata(ctx, owner, repoName)
	if err != nil {
		return fmt.Errorf("failed to get metadata: %w", err)
	}

	// Parse metadata
	var metadata config.Metadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return fmt.Errorf("failed to parse metadata: %w", err)
	}

	// Update metadata:
	// - owner: <newowner>
	// - Change your role from "owner" to "editor"
	// - Change newowner role to "owner"

	metadata.Owner = newOwner

	for i := range metadata.Users {
		if metadata.Users[i].Username == cfg.CurrentUser {
			metadata.Users[i].Role = "editor"
		}
		if metadata.Users[i].Username == newOwner {
			metadata.Users[i].Role = "owner"
		}
	}

	// Upload metadata.json
	ui.PrintInfo("Uploading metadata...")
	updatedMetadataBytes, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if err := client.PutMetadata(ctx, owner, repoName, updatedMetadataBytes); err != nil {
		return fmt.Errorf("failed to upload metadata: %w", err)
	}

	// Update config.json: CurrentUserRole: "editor"
	cfg.CurrentUserRole = "editor"
	cfg.Metadata = &metadata

	if err := cfg.Save(""); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println()
	ui.PrintSuccess("✅ Ownership transferred to %s", newOwner)
	fmt.Println()

	return nil
}

func init() {
	rootCmd.AddCommand(transferOwnerCmd)
}
