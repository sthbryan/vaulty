package main

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"github.com/sthbryan/vaulty/internal/config"
	"github.com/sthbryan/vaulty/internal/session"
	"github.com/sthbryan/vaulty/internal/storage"
	"github.com/sthbryan/vaulty/internal/ui"
	"github.com/sthbryan/vaulty/pkg/application/usecases/auth"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to Vaulty with your credentials",
	Long: `Login to Vaulty by authenticating as a specific user.

This command will:
  • Prompt for username (with suggestion from config if available)
  • Prompt for master password
  • Decrypt your keys and vault
  • Create an active session`,
	RunE: runLogin,
}

func runLogin(cmd *cobra.Command, args []string) error {
	fmt.Println()
	ui.PrintAnimatedLogo()
	fmt.Println()

	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if cfg.Repo == "" && !cfg.IsLocalMode() {
		return fmt.Errorf("Vaulty not initialized. Run 'vty init' first")
	}

	mgr := session.GetManager()
	existingSession := mgr.Get(cfg.CurrentUser)
	if cfg.CurrentUser != "" && existingSession != nil && existingSession.MasterKey != nil {
		relogin, err := ui.AskConfirm(fmt.Sprintf("Already logged in as %s. Re-login with different user?", cfg.CurrentUser), false)
		if err != nil {
			return fmt.Errorf("confirmation failed: %w", err)
		}
		if !relogin {
			fmt.Println("Login cancelled")
			return nil
		}
	}

	var username string
	defaultUsername := ""
	if cfg.Metadata != nil && len(cfg.Metadata.Users) > 0 {
		defaultUsername = cfg.Metadata.Users[0].Username
	}

	err = huh.NewInput().
		Title("Username").
		Placeholder(defaultUsername).
		Value(&username).
		Validate(func(s string) error {
			if s == "" {
				if defaultUsername != "" {
					username = defaultUsername
					return nil
				}
				return fmt.Errorf("username is required")
			}
			return nil
		}).
		Run()
	if err != nil {
		return fmt.Errorf("form cancelled")
	}

	if username == "" {
		username = defaultUsername
	}

	var masterPassword string
	err = huh.NewInput().
		Title("Master password").
		Placeholder("Enter your master password").
		EchoMode(huh.EchoModePassword).
		Value(&masterPassword).
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

	factory := storage.NewFactory(cfg)
	loginUseCase := auth.NewLoginUseCase(factory)

	ctx, cancel := context.WithTimeout(cmd.Context(), 2*time.Minute)
	defer cancel()

	fmt.Println()
	fmt.Println(ui.MutedStyle.Render("Validating credentials..."))

	output, err := loginUseCase.Execute(ctx, auth.LoginInput{
		Username:       username,
		MasterPassword: masterPassword,
	})
	if err != nil {
		fmt.Println()
		fmt.Println(ui.ErrorStyle.Render("❌ Login failed"))
		fmt.Println()
		fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Error: %v", err)))
		fmt.Println()
		return fmt.Errorf("login failed: %w", err)
	}

	fmt.Println(ui.MutedStyle.Render("Creating session..."))

	if err := loginUseCase.SaveSession(output.Session, cfg, output.Metadata, masterPassword); err != nil {
		return fmt.Errorf("saving session: %w", err)
	}

	fmt.Println()
	fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("✅ Logged in as %s (%s)", username, output.Session.Role)))

	return nil
}

func init() {
	rootCmd.AddCommand(loginCmd)
}
