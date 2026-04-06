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

func runInitGitHub(cfg *config.Config) error {
	var repoInput string

	if cfg.Repo != "" {
		fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("Existing vault found: %s", cfg.Repo)))

		useExisting, err := ui.AskConfirm("Use this repository?", true)
		if err != nil {
			return fmt.Errorf("confirmation failed: %w", err)
		}

		if useExisting {
			repoInput = cfg.Repo
		}
	}

	if repoInput == "" {
		var vaultOption string
		err := huh.NewSelect[string]().
			Title("Vault name").
			Description("Choose a name for your vault repository").
			Options(
				huh.NewOption("my-vault (default)", "my-vault"),
				huh.NewOption("Custom name", "custom"),
			).
			Value(&vaultOption).
			Run()
		if err != nil {
			return fmt.Errorf("form cancelled")
		}

		if vaultOption == "custom" {
			err := huh.NewInput().
				Title("Enter vault name").
				Placeholder("my-secrets").
				Value(&repoInput).
				Run()
			if err != nil {
				return fmt.Errorf("form cancelled")
			}
		} else {
			repoInput = vaultOption
		}
	}

	var defaultUsername string
	if cfg.CurrentUser != "" {
		defaultUsername = cfg.CurrentUser
	}

	fmt.Println()
	var owner, repo string

	if cfg.Repo != "" && repoInput == cfg.Repo {
		parts := strings.Split(cfg.Repo, "/")
		if len(parts) >= 2 {
			owner = parts[0]
			repo = parts[1]
		}
	} else {
		err := huh.NewInput().
			Title("GitHub owner/organization").
			Placeholder("your-username or org name").
			Value(&owner).
			Validate(func(s string) error {
				if s == "" {
					return fmt.Errorf("owner is required")
				}
				return nil
			}).
			Run()
		if err != nil {
			return fmt.Errorf("form cancelled")
		}
		repo = repoInput
	}

	if owner == "" || repo == "" {
		return fmt.Errorf("invalid owner or repo")
	}

	var ownerUsername string
	err := huh.NewInput().
		Title("Your username for this vault").
		Placeholder(defaultUsername).
		Value(&ownerUsername).
		Validate(func(s string) error {
			if s == "" {
				if defaultUsername != "" {
					ownerUsername = defaultUsername
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

	if ownerUsername == "" {
		ownerUsername = defaultUsername
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

	cfg.StorageType = "github"
	cfg.Repo = fmt.Sprintf("%s/%s", owner, repo)

	factory := storage.NewFactory(cfg)
	initUseCase := vault.NewInitVaultUseCase(factory)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	fmt.Println()
	fmt.Println(ui.MutedStyle.Render("Initializing vault..."))

	environments, err := selectEnvironments()
	if err != nil {
		return fmt.Errorf("selecting environments: %w", err)
	}

	output, err := initUseCase.ExecuteGitHub(ctx, vault.InitVaultInput{
		Username:     ownerUsername,
		Password:     password1,
		Environments: environments,
	})
	if err != nil {
		return fmt.Errorf("initializing vault: %w", err)
	}

	if err := passStorage.Set(password1); err != nil {
		return fmt.Errorf("storing password: %w", err)
	}

	cfg.SetCurrentUser(ownerUsername, "owner")
	cfg.Metadata = output.Metadata
	cfg.Environments = environments

	if err := cfg.Save(""); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	session.GetManager().Create(output.Session)

	fmt.Println()
	fmt.Println(ui.SuccessStyle.Render("✅ Repository initialized successfully!"))
	fmt.Println()
	fmt.Println()
	fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("Username: %s", ownerUsername)))
	fmt.Println(ui.InfoStyle.Render("Role: owner"))
	fmt.Println()

	return nil
}
