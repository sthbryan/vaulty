package main

import (
	"context"
	"fmt"

	"github.com/DeadBryam/vaulty/internal/cli"
	"github.com/DeadBryam/vaulty/internal/ui"
	"github.com/spf13/cobra"
)

func runDeleteSSH(cmd *cobra.Command, args []string) error {
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
	if deleteUser != "" {
		path = fmt.Sprintf("ssh/%s/%s.vty", deleteUser, name)
	} else {
		path = fmt.Sprintf("ssh/%s.vty", name)
	}

	_, err = s.GetSSHKey(ctx, deleteUser, name)
	if err != nil {
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
		if err := ui.ConfirmOrAbort("Are you sure you want to delete this SSH key?"); err != nil {
			return nil
		}
	}

	ui.PrintInfo("Deleting...")

	if err := s.DeleteSSHKey(ctx, deleteUser, name, ""); err != nil {
		return fmt.Errorf("failed to delete: %w", err)
	}

	fmt.Println()
	ui.PrintSuccess("SSH key deleted successfully!")
	ui.PrintInfo("Name: %s", name)
	ui.PrintInfo("Path: %s", path)
	fmt.Println()

	return nil
}
