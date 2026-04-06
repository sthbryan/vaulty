package main

import (
	"context"
	"fmt"
	"time"

	"github.com/DeadBryam/vaulty/internal/config"
	"github.com/DeadBryam/vaulty/internal/storage"
	"github.com/DeadBryam/vaulty/internal/ui"
	"github.com/DeadBryam/vaulty/pkg/application/usecases/users"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

var (
	addUserRole string
)

var addUserCmd = &cobra.Command{
	Use:   "add-user <username>",
	Short: "Add a new user to the vault",
	Long: `Add a new user to your Vaulty vault.

This command allows the vault owner to add new users with editor or viewer access.
You must provide your master password to verify ownership.

Examples:
  vty add-user juan                    # Add as editor (default)
  vty add-user juan --role viewer      # Add as viewer
  vty add-user juan --role editor      # Add as editor`,
	Args: cobra.ExactArgs(1),
	RunE: runAddUser,
}

func runAddUser(cmd *cobra.Command, args []string) error {
	username := args[0]

	role := addUserRole
	if role != "editor" && role != "viewer" {
		return fmt.Errorf("invalid role: %s (must be 'editor' or 'viewer')", role)
	}

	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if cfg.IsLocalMode() {
		return fmt.Errorf("user management is not supported in local mode. Local vaults are single-owner only")
	}

	if cfg.Repo == "" {
		return fmt.Errorf("no vault initialized - run 'vty init' first")
	}

	if cfg.CurrentUserRole != "owner" {
		return fmt.Errorf("only vault owner can add users")
	}

	fmt.Println()
	fmt.Println(ui.InfoStyle.Render("🔐 Verifying vault ownership"))
	fmt.Println()

	var ownerPassword string
	err = huh.NewInput().
		Title("Your master password").
		Placeholder("Enter your master password").
		EchoMode(huh.EchoModePassword).
		Value(&ownerPassword).
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

	fmt.Println()
	fmt.Println(ui.MutedStyle.Render("Processing..."))

	factory := storage.NewFactory(cfg)
	addUserUseCase := users.NewAddUserUseCase(factory)

	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
	defer cancel()

	var _ *struct{}
	output, err := addUserUseCase.Execute(ctx, users.AddUserInput{
		Username:      username,
		Role:          role,
		OwnerPassword: ownerPassword,
	})
	_ = output
	if err != nil {
		fmt.Println()
		fmt.Println(ui.ErrorStyle.Render("❌ Failed to add user"))
		fmt.Println()
		fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Error: %v", err)))
		fmt.Println()
		return fmt.Errorf("adding user: %w", err)
	}

	fmt.Println()
	fmt.Println(ui.SuccessStyle.Render("✅ User created successfully!"))
	fmt.Println()
	fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("Username: %s", username)))
	fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("Role: %s", role)))
	fmt.Println()

	return nil
}

func init() {
	rootCmd.AddCommand(addUserCmd)
	addUserCmd.Flags().StringVarP(&addUserRole, "role", "r", "editor", "User role (editor or viewer)")
}
