package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/sthbryan/vaulty/internal/config"
	"github.com/sthbryan/vaulty/internal/github"
	"github.com/sthbryan/vaulty/internal/ui"
)

func runDeleteVault(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if cfg.Repo == "" && !cfg.IsLocalMode() {
		return fmt.Errorf("vaulty not initialized. run 'vty init' first")
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
	ui.PrintInfo("  - All user keys")
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

		return deleteLocalVault(cfg)
	}

	return deleteCloudVault(cfg, ctx)
}

func deleteLocalVault(cfg *config.Config) error {
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

	dirsToClean := []string{"envs", "ssh", "resource", "config", ".vaulty"}

	ui.PrintInfo("Deleting vault contents from Vault...")

	for _, dir := range dirsToClean {
		items, err := client.ListDirectory(ctx, owner, repoName, dir)
		if err != nil {
			ui.PrintInfo("  Skipping %s (not found)", dir)
			continue
		}

		for _, item := range items {
			filePath := fmt.Sprintf("%s/%s", dir, item.Name)
			if item.Type == "dir" {
				subItems, _ := client.ListDirectory(ctx, owner, repoName, filePath)
				for _, sub := range subItems {
					subPath := fmt.Sprintf("%s/%s/%s", dir, item.Name, sub.Name)
					err := client.DeleteContent(ctx, owner, repoName, subPath, sub.Sha)
					if err == nil {
						ui.PrintInfo("  Deleted: %s", subPath)
					}
				}
			} else {
				err := client.DeleteContent(ctx, owner, repoName, filePath, item.Sha)
				if err == nil {
					ui.PrintInfo("  Deleted: %s", filePath)
				}
			}
		}
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
	ui.PrintWarning("You can now reinitialize with 'vty init'")
	fmt.Println()

	return nil
}
