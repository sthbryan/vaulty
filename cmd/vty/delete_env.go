package main

import (
	"context"
	"fmt"

	"github.com/DeadBryam/vaulty/internal/cli"
	"github.com/DeadBryam/vaulty/internal/ui"
	"github.com/spf13/cobra"
)

func runDeleteEnv(cmd *cobra.Command, args []string) error {
	name := args[0]

	if err := cli.ValidateName(name); err != nil {
		return err
	}

	s, cfg, err := getStorageForDelete()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	ctx := context.Background()

	var path string
	if deleteEnv != "" {
		path = fmt.Sprintf("envs/%s/%s.vty", deleteEnv, name)
	} else {
		path = fmt.Sprintf("envs/%s.vty", name)
	}

	_, err = s.GetEnv(ctx, deleteEnv, name)
	if err != nil {
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
		if err := ui.ConfirmOrAbort("Are you sure you want to delete this secret?"); err != nil {
			return nil
		}
	}

	ui.PrintInfo("Deleting...")

	if err := s.DeleteEnv(ctx, deleteEnv, name); err != nil {
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

	s, cfg, err := getStorageForDelete()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	ctx := context.Background()

	secrets, err := s.ListEnvSecrets(ctx, deleteEnv)
	if err != nil {
		return fmt.Errorf("failed to list environment: %w", err)
	}

	if len(secrets) == 0 {
		ui.PrintInfo("No secrets found in environment: %s", deleteEnv)
		return nil
	}

	fmt.Println()
	ui.PrintWarning("You are about to delete %d secrets from environment: %s", len(secrets), deleteEnv)
	ui.PrintInfo("Repository: %s", cfg.Repo)
	fmt.Println()

	for _, secret := range secrets {
		ui.PrintInfo("  - %s", secret)
	}
	fmt.Println()

	if !deleteForce {
		if err := ui.ConfirmOrAbort(fmt.Sprintf("Are you sure you want to delete %d secrets from %s?", len(secrets), deleteEnv)); err != nil {
			return nil
		}
	}

	ui.PrintInfo("Deleting secrets...")
	deletedCount := 0

	for _, secret := range secrets {
		err := s.DeleteEnv(ctx, deleteEnv, secret)
		if err != nil {
			logger.Warn("failed to delete secret", "name", secret, "error", err)
			continue
		}
		deletedCount++
		ui.PrintInfo("  Deleted: %s", secret)
	}

	fmt.Println()
	ui.PrintSuccess("Deleted %d/%d secrets from environment: %s", deletedCount, len(secrets), deleteEnv)
	fmt.Println()

	return nil
}
