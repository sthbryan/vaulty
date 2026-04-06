package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/sthbryan/vaulty/internal/config"
	"github.com/sthbryan/vaulty/internal/password"
	"github.com/sthbryan/vaulty/internal/session"
	"github.com/sthbryan/vaulty/internal/storage"
	"github.com/sthbryan/vaulty/internal/ui"
	"github.com/sthbryan/vaulty/pkg/application/usecases/vault"
)

func selectEnvironments() ([]string, error) {
	var selectedEnvs []string

	err := huh.NewMultiSelect[string]().
		Title("Select environments (select multiple)").
		Options(
			huh.NewOption("production", "production"),
			huh.NewOption("staging", "staging"),
			huh.NewOption("development", "development"),
			huh.NewOption("test", "test"),
			huh.NewOption("local", "local"),
		).
		Value(&selectedEnvs).
		Run()
	if err != nil {
		return nil, fmt.Errorf("form cancelled")
	}

	var customOption string
	err = huh.NewSelect[string]().
		Title("Add custom environments?").
		Options(
			huh.NewOption("No", "no"),
			huh.NewOption("Yes", "yes"),
		).
		Value(&customOption).
		Run()
	if err != nil {
		return nil, fmt.Errorf("form cancelled")
	}

	if customOption == "yes" {
		var customInput string
		err = huh.NewInput().
			Title("Enter custom environments (comma-separated)").
			Placeholder("qa, uat, demo").
			Value(&customInput).
			Run()
		if err != nil {
			return nil, fmt.Errorf("form cancelled")
		}

		envParts := strings.Split(customInput, ",")
		for _, e := range envParts {
			e = strings.TrimSpace(e)
			if e != "" {
				exists := false
				for _, existing := range selectedEnvs {
					if existing == e {
						exists = true
						break
					}
				}
				if !exists {
					selectedEnvs = append(selectedEnvs, e)
				}
			}
		}
	}

	if len(selectedEnvs) == 0 {
		selectedEnvs = []string{"production"}
	}

	return selectedEnvs, nil
}

func runInitLocal(cfg *config.Config) error {
	var username string

	fmt.Println()
	var defaultUsername string
	if cfg.CurrentUser != "" {
		defaultUsername = cfg.CurrentUser
	}

	err := huh.NewInput().
		Title("Your username for this vault").
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

	environments, err := selectEnvironments()
	if err != nil {
		return fmt.Errorf("selecting environments: %w", err)
	}

	fmt.Println()
	fmt.Println(ui.InfoStyle.Render("🔐 Create your master password"))
	fmt.Println()

	var password1, password2 string

	err = huh.NewInput().
		Title("Master password").
		Placeholder("Enter a strong password").
		EchoMode(huh.EchoModePassword).
		Value(&password1).
		Validate(func(s string) error {
			if s == "" {
				return fmt.Errorf("password cannot be empty")
			}
			if strings.Contains(s, " ") {
				return fmt.Errorf("password cannot contain spaces")
			}
			if len(s) < 8 {
				return fmt.Errorf("password must be at least 8 characters")
			}
			return nil
		}).
		Run()
	if err != nil {
		return fmt.Errorf("form cancelled")
	}

	err = huh.NewInput().
		Title("Confirm password").
		Placeholder("Re-enter your password").
		EchoMode(huh.EchoModePassword).
		Value(&password2).
		Validate(func(s string) error {
			if s == "" {
				return fmt.Errorf("please confirm your password")
			}
			if s != password1 {
				return fmt.Errorf("passwords do not match")
			}
			return nil
		}).
		Run()
	if err != nil {
		return fmt.Errorf("form cancelled")
	}

	passStorage, err := password.NewStorage()
	if err != nil {
		return fmt.Errorf("password storage: %w", err)
	}

	cfg.StorageType = "local"
	cfg.LocalVaultPath = ""

	factory := storage.NewFactory(cfg)
	initUseCase := vault.NewInitVaultUseCase(factory)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	fmt.Println()
	fmt.Println(ui.MutedStyle.Render("Initializing local vault..."))

	if err != nil {
	}

	output, err := initUseCase.ExecuteLocal(ctx, vault.InitVaultInput{
		Username:     username,
		Password:     password1,
		Environments: environments,
	})
	if err != nil {
		return fmt.Errorf("initializing vault: %w", err)
	}

	if err := passStorage.Set(password1); err != nil {
		return fmt.Errorf("storing password: %w", err)
	}

	cfg.SetCurrentUser(username, "owner")
	cfg.Metadata = output.Metadata
	cfg.Environments = environments

	if err := cfg.Save(""); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	session.GetManager().Create(output.Session)

	fmt.Println()
	fmt.Println(ui.SuccessStyle.Render("✅ Local vault initialized successfully!"))
	fmt.Println()
	fmt.Println()
	fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("Username: %s", username)))
	fmt.Println(ui.InfoStyle.Render("Role: owner"))
	fmt.Println()

	return nil
}
