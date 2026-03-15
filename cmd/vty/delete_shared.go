package main

import (
	"context"
	"fmt"
	"os"

	"github.com/DeadBryam/vaulty/internal/config"
	"github.com/DeadBryam/vaulty/internal/github"
	"github.com/DeadBryam/vaulty/internal/ui"
	"github.com/spf13/cobra"
)

func runDeleteVault(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if cfg.Repo == "" && !cfg.IsLocalMode() {
		return fmt.Errorf("Vaulty not initialized. Run 'vty init' first")
	}

	if !cfg.IsOwner() {
		return fmt.Errorf("only the vault owner can delete the entire vault")
	}

	fmt.Println()
	ui.PrintWarning("⚠️  DESTRUCTIVE OPERATION ⚠️")
	fmt.Println()
	ui.PrintWarning("You are about to delete the ENTIRE VAULT!")
	ui.PrintWarning("This will permanently remove:")
	ui.PrintInfo("  - All environment secrets (shared and all environments)")
	ui.PrintInfo("  - All SSH keys")
	ui.PrintInfo("  - All user keys and recovery files")
	ui.PrintInfo("  - Vault metadata")
	ui.PrintInfo("")
	ui.PrintWarning("This action CANNOT be undone!")
	ui.PrintInfo("")

	if cfg.IsLocalMode() {
		ui.PrintInfo("Storage: Local (%s)", cfg.LocalVaultPath)
	} else {
		ui.PrintInfo("Repository: %s", cfg.Repo)
	}
	fmt.Println()

	if !deleteForce {
		confirmed, err := ui.AskConfirm("Are you ABSOLUTELY sure you want to delete the entire vault?", false)
		if err != nil {
			return fmt.Errorf("confirmation failed: %w", err)
		}
		if !confirmed {
			ui.PrintInfo("Delete cancelled")
			return nil
		}
	}

	ctx := context.Background()

	if cfg.IsLocalMode() {

		return deleteLocalVault(cfg, ctx)
	}

	return deleteCloudVault(cfg, ctx)
}

func deleteLocalVault(cfg *config.Config, ctx context.Context) error {
	vaultPath := cfg.LocalVaultPath
	if vaultPath == "" {
		vaultPath = cfg.DefaultLocalVaultPath()
	}

	ui.PrintInfo("Deleting local vault from %s...", vaultPath)

	err := os.RemoveAll(vaultPath)
	if err != nil {
		return fmt.Errorf("failed to delete vault directory: %w", err)
	}

	cfg.SetRepo("")
	cfg.StorageType = ""
	cfg.LocalVaultPath = ""
	cfg.CurrentUser = ""
	cfg.CurrentUserRole = ""
	cfg.Metadata = nil

	if err := cfg.Save(""); err != nil {
		return fmt.Errorf("failed to clear config: %w", err)
	}

	fmt.Println()
	ui.PrintSuccess("Vault deleted successfully!")
	ui.PrintInfo("Deleted local vault from: %s", vaultPath)
	ui.PrintWarning("You can now reinitialize with 'vty init' or 'vty init --local'")
	fmt.Println()

	return nil
}

func deleteCloudVault(cfg *config.Config, ctx context.Context) error {
	token, err := github.GetGitHubToken()
	if err != nil {
		return fmt.Errorf("GitHub authentication: %w", err)
	}

	client := github.NewClient(token)
	owner, repoName, err := github.ParseRepo(cfg.Repo)
	if err != nil {
		return fmt.Errorf("invalid repo format: %w", err)
	}

	pathsToDelete := []string{".vaulty", "envs", "ssh"}

	ui.PrintInfo("Deleting vault contents from GitHub...")
	deletedCount := 0

	for _, path := range pathsToDelete {

		content, err := client.GetContent(ctx, owner, repoName, path)
		if err != nil || content == nil {
			ui.PrintInfo("  Skipping %s (not found)", path)
			continue
		}

		err = client.DeleteContent(ctx, owner, repoName, path, content.Sha)
		if err != nil {
			logger.Warn("failed to delete", "path", path, "error", err)
			continue
		}
		deletedCount++
		ui.PrintInfo("  Deleted: %s", path)
	}

	cfg.SetRepo("")
	cfg.StorageType = ""
	cfg.CurrentUser = ""
	cfg.CurrentUserRole = ""
	cfg.Metadata = nil

	if err := cfg.Save(""); err != nil {
		return fmt.Errorf("failed to clear config: %w", err)
	}

	fmt.Println()
	ui.PrintSuccess("Vault deleted successfully!")
	ui.PrintInfo("Deleted %d/%d items", deletedCount, len(pathsToDelete))
	ui.PrintWarning("You can now reinitialize with 'vty init'")
	fmt.Println()

	return nil
}
