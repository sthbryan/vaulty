package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/DeadBryam/vaulty/internal/config"
	"github.com/DeadBryam/vaulty/internal/github"
	"github.com/DeadBryam/vaulty/internal/ui"
)

var (
	deleteForce bool
	deleteType  string
)

var deleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a secret from the vault",
	Long: `Delete a secret from your Vaulty repository.

This command will permanently remove the secret from GitHub.
This action cannot be undone.

Examples:
  vty delete my-secret
  vty delete my-secret --force
  vty delete my-secret --type=env
  vty delete my-secret --type=ssh`,
	Args: cobra.ExactArgs(1),
	RunE: runDelete,
}

func runDelete(cmd *cobra.Command, args []string) error {
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

	pathsToTry := []string{}
	if deleteType == "" || deleteType == "env" {
		pathsToTry = append(pathsToTry, fmt.Sprintf("envs/%s.vty", name))
	}
	if deleteType == "" || deleteType == "ssh" {
		pathsToTry = append(pathsToTry, fmt.Sprintf("ssh/%s.vty", name))
	}

	var foundPath string
	var foundSha string

	for _, path := range pathsToTry {
		content, err := client.GetContent(ctx, owner, repoName, path)
		if err == nil && content != nil {
			foundPath = path
			foundSha = content.Sha
			break
		}
	}

	if foundPath == "" {
		return fmt.Errorf("secret not found: %s", name)
	}

	fmt.Println()
	ui.PrintWarning("You are about to delete: %s", name)
	ui.PrintInfo("Path: %s", foundPath)
	ui.PrintInfo("Repository: %s", cfg.Repo)
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

	if err := client.DeleteContent(ctx, owner, repoName, foundPath, foundSha); err != nil {
		return fmt.Errorf("failed to delete: %w", err)
	}

	fmt.Println()
	ui.PrintSuccess("Secret deleted successfully!")
	ui.PrintInfo("Name: %s", name)
	ui.PrintInfo("Path: %s", foundPath)
	fmt.Println()

	return nil
}

func init() {
	deleteCmd.Flags().BoolVarP(&deleteForce, "force", "f", false, "Force delete without confirmation")
	deleteCmd.Flags().StringVar(&deleteType, "type", "", "Secret type: env, ssh (auto-detect if not specified)")
}
