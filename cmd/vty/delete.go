package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/DeadBryam/vaulty/internal/config"
	"github.com/DeadBryam/vaulty/internal/github"
	"github.com/DeadBryam/vaulty/internal/ui"
	"github.com/spf13/cobra"
)

var (
	deleteForce bool
	deleteEnv   string
	deleteUser  string
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete secrets, environments, or vault",
	Long: `Delete secrets, environments, or the entire vault.

Examples:
  vty delete env <name>              # Delete secret from shared
  vty delete env <name> --env=staging # Delete secret from environment
  vty delete envs --env=staging       # Delete all secrets from environment
  vty delete ssh <name>               # Delete SSH key
  vty delete vault                     # Delete entire vault (owner only)`,
}

var deleteEnvCmd = &cobra.Command{
	Use:   "env <name>",
	Short: "Delete a specific environment variable",
	Long: `Delete a specific environment variable from the vault.

If --env is not specified, deletes from shared (envs/{name}.vty).
If --env is specified, deletes from envs/{env}/{name}.vty.`,
	Args: cobra.ExactArgs(1),
	RunE: runDeleteEnv,
}

var deleteEnvsCmd = &cobra.Command{
	Use:   "envs",
	Short: "Delete all secrets from an environment",
	Long: `Delete all secrets from a specific environment.

This will permanently remove all secrets from the specified environment.
Use with caution - this action cannot be undone.`,
	RunE: runDeleteEnvs,
}

var deleteSSHCmd = &cobra.Command{
	Use:   "ssh <name>",
	Short: "Delete an SSH key",
	Long: `Delete an SSH key from the vault.

Examples:
  vty delete ssh my-key
  vty delete ssh my-key -u username`,
	Args: cobra.ExactArgs(1),
	RunE: runDeleteSSH,
}

var deleteVaultCmd = &cobra.Command{
	Use:   "vault",
	Short: "Delete entire vault (DESTRUCTIVE - owner only)",
	Long: `Delete the entire vault including all secrets, SSH keys, and users.

This is a DESTRUCTIVE operation that will:
  - Delete all environment secrets
  - Delete all SSH keys
  - Delete all user keys and recovery files
  - Delete metadata

This action CANNOT be undone. Only the vault owner can perform this action.`,
	RunE: runDeleteVault,
}

func runDeleteEnv(cmd *cobra.Command, args []string) error {
	name := args[0]

	if strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return fmt.Errorf("name cannot contain path separators")
	}

	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

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

	var path string
	if deleteEnv != "" {
		path = fmt.Sprintf("envs/%s/%s.vty", deleteEnv, name)
	} else {
		path = fmt.Sprintf("envs/%s.vty", name)
	}

	content, err := client.GetContent(ctx, owner, repoName, path)
	if err != nil || content == nil {
		return fmt.Errorf("secret not found: %s", name)
	}

	fmt.Println()
	ui.PrintWarning("You are about to delete: %s", name)
	ui.PrintInfo("Path: %s", path)
	ui.PrintInfo("Repository: %s", cfg.Repo)
	if deleteEnv != "" {
		ui.PrintInfo("Environment: %s", deleteEnv)
	} else {
		ui.PrintInfo("Environment: shared")
	}
	fmt.Println()

	if !deleteForce {
		confirmed, err := ui.AskConfirm("Are you sure you want to delete this secret?", false)
		if err != nil {
			return fmt.Errorf("confirmation failed: %w", err)
		}
		if !confirmed {
			ui.PrintInfo("Delete cancelled")
			return nil
		}
	}

	ui.PrintInfo("Deleting from GitHub...")

	if err := client.DeleteContent(ctx, owner, repoName, path, content.Sha); err != nil {
		return fmt.Errorf("failed to delete: %w", err)
	}

	fmt.Println()
	ui.PrintSuccess("Secret deleted successfully!")
	ui.PrintInfo("Name: %s", name)
	ui.PrintInfo("Path: %s", path)
	fmt.Println()

	return nil
}

func runDeleteEnvs(cmd *cobra.Command, args []string) error {

	if deleteEnv == "" {
		return fmt.Errorf("the --env flag is required for this command")
	}

	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

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

	envPath := fmt.Sprintf("envs/%s", deleteEnv)
	items, err := client.ListDirectory(ctx, owner, repoName, envPath)
	if err != nil {
		return fmt.Errorf("failed to list environment: %w", err)
	}

	var secretsToDelete []struct {
		name string
		path string
		sha  string
	}

	for _, item := range items {
		if strings.HasSuffix(item.Name, ".vty") {
			secretsToDelete = append(secretsToDelete, struct {
				name string
				path string
				sha  string
			}{
				name: strings.TrimSuffix(item.Name, ".vty"),
				path: fmt.Sprintf("%s/%s", envPath, item.Name),
				sha:  item.Sha,
			})
		}
	}

	if len(secretsToDelete) == 0 {
		ui.PrintInfo("No secrets found in environment: %s", deleteEnv)
		return nil
	}

	fmt.Println()
	ui.PrintWarning("You are about to delete %d secrets from environment: %s", len(secretsToDelete), deleteEnv)
	ui.PrintInfo("Repository: %s", cfg.Repo)
	fmt.Println()

	for _, secret := range secretsToDelete {
		ui.PrintInfo("  - %s", secret.name)
	}
	fmt.Println()

	if !deleteForce {
		confirmed, err := ui.AskConfirm(fmt.Sprintf("Are you sure you want to delete %d secrets from %s?", len(secretsToDelete), deleteEnv), false)
		if err != nil {
			return fmt.Errorf("confirmation failed: %w", err)
		}
		if !confirmed {
			ui.PrintInfo("Delete cancelled")
			return nil
		}
	}

	ui.PrintInfo("Deleting secrets from GitHub...")
	deletedCount := 0

	for _, secret := range secretsToDelete {
		err := client.DeleteContent(ctx, owner, repoName, secret.path, secret.sha)
		if err != nil {
			logger.Warn("failed to delete secret", "name", secret.name, "error", err)
			continue
		}
		deletedCount++
		ui.PrintInfo("  Deleted: %s", secret.name)
	}

	fmt.Println()
	ui.PrintSuccess("Deleted %d/%d secrets from environment: %s", deletedCount, len(secretsToDelete), deleteEnv)
	fmt.Println()

	return nil
}

func runDeleteSSH(cmd *cobra.Command, args []string) error {
	name := args[0]

	if strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return fmt.Errorf("name cannot contain path separators")
	}

	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

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

	var path string
	if deleteUser != "" {
		path = fmt.Sprintf("ssh/%s/%s.vty", deleteUser, name)
	} else {
		path = fmt.Sprintf("ssh/%s.vty", name)
	}

	content, err := client.GetContent(ctx, owner, repoName, path)
	if err != nil || content == nil {
		return fmt.Errorf("SSH key not found: %s", name)
	}

	fmt.Println()
	ui.PrintWarning("You are about to delete SSH key: %s", name)
	ui.PrintInfo("Path: %s", path)
	ui.PrintInfo("Repository: %s", cfg.Repo)
	if deleteUser != "" {
		ui.PrintInfo("User: %s", deleteUser)
	}
	fmt.Println()

	if !deleteForce {
		confirmed, err := ui.AskConfirm("Are you sure you want to delete this SSH key?", false)
		if err != nil {
			return fmt.Errorf("confirmation failed: %w", err)
		}
		if !confirmed {
			ui.PrintInfo("Delete cancelled")
			return nil
		}
	}

	ui.PrintInfo("Deleting from GitHub...")

	if err := client.DeleteContent(ctx, owner, repoName, path, content.Sha); err != nil {
		return fmt.Errorf("failed to delete: %w", err)
	}

	fmt.Println()
	ui.PrintSuccess("SSH key deleted successfully!")
	ui.PrintInfo("Name: %s", name)
	ui.PrintInfo("Path: %s", path)
	fmt.Println()

	return nil
}

func runDeleteVault(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("configuration error: %w", err)
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
	ui.PrintInfo("Repository: %s", cfg.Repo)
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

	fmt.Println()
	ui.PrintSuccess("Vault deleted successfully!")
	ui.PrintInfo("Deleted %d/%d items", deletedCount, len(pathsToDelete))
	ui.PrintWarning("You can now unlink or reinitialize the vault with 'vty unlink' or 'vty init'")
	fmt.Println()

	return nil
}

func init() {

	deleteCmd.AddCommand(deleteEnvCmd)
	deleteCmd.AddCommand(deleteEnvsCmd)
	deleteCmd.AddCommand(deleteSSHCmd)
	deleteCmd.AddCommand(deleteVaultCmd)

	rootCmd.AddCommand(deleteCmd)

	deleteEnvCmd.Flags().StringVar(&deleteEnv, "env", "", "Environment (optional)")
	deleteEnvCmd.Flags().BoolVarP(&deleteForce, "force", "f", false, "Force delete without confirmation")

	deleteEnvsCmd.Flags().StringVar(&deleteEnv, "env", "", "Environment (required)")
	deleteEnvsCmd.Flags().BoolVarP(&deleteForce, "force", "f", false, "Force delete without confirmation")

	deleteSSHCmd.Flags().StringVarP(&deleteUser, "user", "u", "", "User (optional)")
	deleteSSHCmd.Flags().BoolVarP(&deleteForce, "force", "f", false, "Force delete without confirmation")

	deleteVaultCmd.Flags().BoolVarP(&deleteForce, "force", "f", false, "Force delete without confirmation")
}
