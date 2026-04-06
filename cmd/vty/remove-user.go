package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/sthbryan/vaulty/internal/config"
	"github.com/sthbryan/vaulty/internal/password"
	"github.com/sthbryan/vaulty/internal/storage"
	"github.com/sthbryan/vaulty/internal/ui"
	"github.com/sthbryan/vaulty/pkg/application/usecases/users"
)

var removeUserCmd = &cobra.Command{
	Use:   "remove-user <username>",
	Short: "Remove a user from the vault and rotate the master key",
	Long: `Remove a user from Vaulty and rotate the master key.

This command will:
  1. Verify you are the vault owner
  2. Decrypt the current vault with the old master key
  3. Generate a new master key
  4. Re-encrypt the vault with the new key
  5. Re-encrypt the new key for all remaining users
  6. Upload all changes to GitHub
  7. Delete the removed user's key file

This action is irreversible. The removed user will no longer have access to the vault.`,
	Args: cobra.ExactArgs(1),
	RunE: runRemoveUser,
}

func runRemoveUser(cmd *cobra.Command, args []string) error {
	username := args[0]

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
		return fmt.Errorf("only the vault owner can remove users")
	}

	if username == cfg.CurrentUser {
		return fmt.Errorf("cannot remove yourself from the vault")
	}

	fmt.Println()
	ui.PrintWarning("You are about to remove: %s", username)
	fmt.Println()

	confirmed, err := ui.AskConfirm("Remove "+username+" from vault? This will rotate the masterKey.", false)
	if err != nil {
		return fmt.Errorf("confirmation failed: %w", err)
	}
	if !confirmed {
		ui.PrintInfo("Remove cancelled")
		return nil
	}

	pwd, err := password.NewStorage()
	if err != nil {
		return fmt.Errorf("failed to create password storage: %w", err)
	}

	currentPassword, err := pwd.Get()
	if err != nil {
		return fmt.Errorf("password not found, run 'vty init")
	}

	verifyPassword, err := ui.AskPassword("Verify your password")
	if err != nil {
		return fmt.Errorf("password prompt failed: %w", err)
	}

	if verifyPassword != currentPassword {
		return fmt.Errorf("password is incorrect")
	}

	factory := storage.NewFactory(cfg)
	removeUserUseCase := users.NewRemoveUserUseCase(factory)

	ctx := context.Background()

	fmt.Println()
	fmt.Println(ui.MutedStyle.Render("Processing..."))

	output, err := removeUserUseCase.Execute(ctx, users.RemoveUserInput{
		Username:      username,
		OwnerPassword: currentPassword,
	})
	if err != nil {
		fmt.Println()
		fmt.Println(ui.ErrorStyle.Render("❌ Failed to remove user"))
		fmt.Println()
		fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Error: %v", err)))
		fmt.Println()
		return fmt.Errorf("removing user: %w", err)
	}

	fmt.Println()
	ui.PrintSuccess("%s removed, masterKey rotated, all users re-encrypted", output.RemovedUser)
	fmt.Println()

	return nil
}

func init() {
	rootCmd.AddCommand(removeUserCmd)
}
