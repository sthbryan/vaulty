package main

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/DeadBryam/vaulty/internal/config"
	"github.com/DeadBryam/vaulty/internal/crypto"
	"github.com/DeadBryam/vaulty/internal/github"
	"github.com/DeadBryam/vaulty/internal/storage"
	"github.com/DeadBryam/vaulty/internal/ui"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

var linkLocalMode bool

var linkCmd = &cobra.Command{
	Use:   "link",
	Short: "Link to an existing Vaulty vault",
	Long: `Link your machine to an existing Vaulty vault.

This command will:
  • Fetch the vault metadata from the specified repository or local storage
  • Store the configuration locally
  • Prepare you to login with 'vty login'

Use --local for local storage mode.`,
	RunE: runLink,
}

func runLink(cmd *cobra.Command, args []string) error {
	fmt.Println()
	ui.PrintAnimatedLogo()
	fmt.Println()

	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if linkLocalMode {
		return runLinkLocal(cfg)
	}

	return runLinkGitHub(cfg)
}

func runLinkGitHub(cfg *config.Config) error {
	var repoFull string

	defaultRepo := ""
	if cfg.Repo != "" {
		defaultRepo = cfg.Repo
	}

	if defaultRepo == "" {
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
				Value(&vaultOption).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("vault name is required")
					}
					if strings.Contains(s, " ") {
						return fmt.Errorf("vault name cannot contain spaces")
					}
					if !regexp.MustCompile(`^[a-zA-Z0-9\-_]+$`).MatchString(s) {
						return fmt.Errorf("vault name can only contain letters, numbers, hyphens and underscores")
					}
					return nil
				}).
				Run()
			if err != nil {
				return fmt.Errorf("form cancelled")
			}
		}

		var ownerInput string
		err = huh.NewInput().
			Title("GitHub owner/organization").
			Placeholder("your-username or org name").
			Value(&ownerInput).
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

		repoFull = ownerInput + "/" + vaultOption
	} else {
		var err3 error
		err3 = huh.NewInput().
			Title("GitHub Repository").
			Placeholder(defaultRepo).
			Value(&repoFull).
			Validate(func(s string) error {
				if s == "" {
					repoFull = defaultRepo
					return nil
				}
				return nil
			}).
			Run()
		if err3 != nil {
			return fmt.Errorf("form cancelled")
		}

		if repoFull == "" {
			repoFull = defaultRepo
		}
	}

	token, err := github.GetGitHubToken()
	if err != nil {
		return fmt.Errorf("GitHub authentication: %w", err)
	}

	client := github.NewClient(token)
	owner, repo, err := github.ParseRepo(repoFull)
	if err != nil {
		return fmt.Errorf("invalid repository format: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println()
	fmt.Println(ui.MutedStyle.Render("Fetching vault metadata..."))

	metadataResp, err := client.GetContent(ctx, owner, repo, ".vaulty/metadata.vty")
	if err != nil {
		return fmt.Errorf("fetching vault metadata: %w", err)
	}

	metadataEncData, err := client.DecodeContent(metadataResp)
	if err != nil {
		return fmt.Errorf("decoding metadata: %w", err)
	}

	metadataJSON, err := crypto.DecompressHex(string(metadataEncData))
	if err != nil {
		return fmt.Errorf("decompressing metadata: %w", err)
	}

	var metadata config.Metadata
	if err := json.Unmarshal(metadataJSON, &metadata); err != nil {
		return fmt.Errorf("parsing metadata: %w", err)
	}

	cfg.SetRepo(repoFull)
	cfg.Metadata = &metadata

	if err := cfg.Save(""); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Println()
	fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("✅ Linked to %s", repoFull)))
	fmt.Println()
	fmt.Println(ui.MutedStyle.Render("Found users:"))
	for _, user := range metadata.Users {
		fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("  • %s (%s)", user.Username, user.Role)))
	}
	fmt.Println()
	fmt.Println(ui.MutedStyle.Render("Run 'vty login' to authenticate"))

	return nil
}

func runLinkLocal(cfg *config.Config) error {
	fmt.Println()
	fmt.Println(ui.MutedStyle.Render("Linking to local vault in ~/.vty/vault/..."))
	fmt.Println()

	cfg.SetLocalMode()
	factory := storage.NewFactory(cfg)
	vaultStorage, err := factory.CreateVaultStorage()
	if err != nil {
		return fmt.Errorf("creating storage: %w", err)
	}

	ctx := context.Background()

	metadataBytes, err := vaultStorage.GetMetadata(ctx)
	if err != nil {
		return fmt.Errorf("fetching vault metadata: %w", err)
	}

	var metadata config.Metadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return fmt.Errorf("parsing metadata: %w", err)
	}

	cfg.SetLocalMode()
	cfg.Metadata = &metadata

	if err := cfg.Save(""); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Println()
	fmt.Println(ui.SuccessStyle.Render("✅ Linked to local vault"))
	fmt.Println()
	fmt.Println(ui.MutedStyle.Render("Found users:"))
	for _, user := range metadata.Users {
		fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("  • %s (%s)", user.Username, user.Role)))
	}
	fmt.Println()
	fmt.Println(ui.MutedStyle.Render("Run 'vty login' to authenticate"))

	return nil
}

func init() {
	rootCmd.AddCommand(linkCmd)
	linkCmd.Flags().BoolVarP(&linkLocalMode, "local", "l", false, "Use local storage instead of GitHub")
}
