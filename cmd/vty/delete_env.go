package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/DeadBryam/vaulty/internal/github"
	"github.com/DeadBryam/vaulty/internal/ui"
	"github.com/spf13/cobra"
)

func runDeleteEnv(cmd *cobra.Command, args []string) error {
	name := args[0]

	if strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return fmt.Errorf("name cannot contain path separators")
	}

	cfg, client, err := getConfigAndClient()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

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

	cfg, client, err := getConfigAndClient()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

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
