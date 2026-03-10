package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/sthbryan/vaulty/internal/config"
	"github.com/sthbryan/vaulty/internal/crypto"
	"github.com/sthbryan/vaulty/internal/github"
	"github.com/sthbryan/vaulty/internal/password"
	"github.com/sthbryan/vaulty/internal/ui"
)

const logo = `
██╗   ██╗ █████╗ ██╗   ██╗██╗  ████████╗██╗   ██╗
██║   ██║██╔══██╗██║   ██║██║  ╚══██╔══╝╚██╗ ██╔╝
██║   ██║███████║██║   ██║██║     ██║    ╚████╔╝ 
╚██╗ ██╔╝██╔══██║██║   ██║██║     ██║     ╚██╔╝  
 ╚████╔╝ ██║  ██║╚██████╔╝███████╗██║      ██║   
  ╚═══╝  ╚═╝  ╚═╝ ╚═════╝ ╚══════╝╚═╝      ╚═╝   
`

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Vaulty with a GitHub repository",
	Long: `Initialize Vaulty by creating or linking a GitHub repository.

This command will guide you through:
  • Setting up a secure master password
  • Linking your GitHub repository
  • Creating a recovery seed phrase`,
	RunE: runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	fmt.Println()
	fmt.Println(lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.Primary)).
		Bold(true).
		Render(logo))
	fmt.Println(ui.TitleStyle.Render("✨ Welcome to Vaulty!"))
	fmt.Println(ui.MutedStyle.Render("  Secure secret management powered by GitHub"))
	fmt.Println()

	var repoInput string
	err := huh.NewInput().
		Title("Repository").
		Placeholder("owner/repo").
		Value(&repoInput).
		Validate(func(s string) error {
			if s == "" {
				return fmt.Errorf("repository is required")
			}
			_, _, err := github.ParseRepo(s)
			return err
		}).
		Run()
	if err != nil {
		return fmt.Errorf("form cancelled")
	}

	owner, repo, _ := github.ParseRepo(repoInput)
	repoFull := fmt.Sprintf("%s/%s", owner, repo)

	token, err := github.GetGitHubToken()
	if err != nil {
		return fmt.Errorf("GitHub authentication: %w", err)
	}

	client := github.NewClient(token)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println()
	fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Checking repository %s...", repoFull)))

	canaryResp, err := client.GetContent(ctx, owner, repo, ".vaulty/canary.vty")
	isNewRepo := err != nil

	cfg := &config.Config{}
	cfg.SetRepo(repoFull)

	passStorage, err := password.NewStorage()
	if err != nil {
		return fmt.Errorf("password storage: %w", err)
	}

	if isNewRepo {
		if err := initializeNewRepo(ctx, client, owner, repo, cfg, passStorage); err != nil {
			return err
		}
	} else {
		if err := linkExistingRepo(ctx, client, owner, repo, canaryResp, cfg, passStorage); err != nil {
			return err
		}
	}

	if err := cfg.Save(""); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	return nil
}

func initializeNewRepo(ctx context.Context, client *github.Client, owner, repo string, cfg *config.Config, passStorage password.Storage) error {
	fmt.Println(ui.InfoStyle.Render("📦 Creating repository..."))

	_, err := client.ListDirectory(ctx, owner, repo, "")
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Repository %s/%s does not exist, creating...", owner, repo)))
			if err := createGitHubRepo(repo); err != nil {
				return fmt.Errorf("creating repository: %w", err)
			}
			time.Sleep(2 * time.Second)
		} else {
			return fmt.Errorf("checking repository: %w", err)
		}
	}

	fmt.Println(ui.InfoStyle.Render("🔐 Create your master password"))
	fmt.Println()

	var password1, password2 string

	err = huh.NewInput().
		Title("Master password").
		Placeholder("Enter a strong password").
		EchoMode(huh.EchoModePassword).
		Value(&password1).
		Validate(func(s string) error {
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
			if s != password1 {
				return fmt.Errorf("passwords do not match")
			}
			return nil
		}).
		Run()
	if err != nil {
		return fmt.Errorf("form cancelled")
	}

	deviceSalt := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, deviceSalt); err != nil {
		return fmt.Errorf("generating device salt: %w", err)
	}
	cfg.DeviceSalt = deviceSalt

	canary, err := crypto.GenerateCanary(password1, deviceSalt)
	if err != nil {
		return fmt.Errorf("generating canary: %w", err)
	}

	canaryContent := base64.StdEncoding.EncodeToString(canary)
	err = client.PutContent(ctx, owner, repo, ".vaulty/canary.vty", github.ContentRequest{
		Message: "Initialize Vaulty repository",
		Content: canaryContent,
	})
	if err != nil {
		return fmt.Errorf("uploading canary: %w", err)
	}

	seedPhrase, err := crypto.GenerateRecoverySeed()
	if err != nil {
		return fmt.Errorf("generating recovery seed: %w", err)
	}

	if err := passStorage.Set(password1); err != nil {
		return fmt.Errorf("storing password: %w", err)
	}

	fmt.Println()
	fmt.Println(ui.SuccessStyle.Render("✅ Repository initialized successfully!"))
	fmt.Println()
	fmt.Println(ui.WarningStyle.Render("⚠️  IMPORTANT: Save your recovery seed phrase"))
	fmt.Println()
	fmt.Println(ui.InfoStyle.Render("Recovery seed phrase:"))
	fmt.Println(lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.Warning)).
		Bold(true).
		Render(seedPhrase))
	fmt.Println()
	fmt.Println(ui.MutedStyle.Render("Write this down and store it securely. You will need it to recover"))
	fmt.Println(ui.MutedStyle.Render("your vault if you forget your master password."))
	fmt.Println()

	return nil
}

func linkExistingRepo(ctx context.Context, client *github.Client, owner, repo string, canaryResp *github.ContentResponse, cfg *config.Config, passStorage password.Storage) error {
	fmt.Println(ui.InfoStyle.Render("🔗 Linking existing repository"))
	fmt.Println()

	canaryData, err := client.DecodeContent(canaryResp)
	if err != nil {
		return fmt.Errorf("decoding canary: %w", err)
	}

	var password string
	err = huh.NewInput().
		Title("Master password").
		Placeholder("Enter your master password").
		EchoMode(huh.EchoModePassword).
		Value(&password).
		Run()
	if err != nil {
		return fmt.Errorf("form cancelled")
	}

	deviceSalt := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, deviceSalt); err != nil {
		return fmt.Errorf("generating device salt: %w", err)
	}
	cfg.DeviceSalt = deviceSalt

	if err := crypto.ValidateCanary(canaryData, password, deviceSalt); err != nil {
		fmt.Println()
		fmt.Println(ui.ErrorStyle.Render("❌ Wrong password. Use 'vty recover' if forgotten."))
		return fmt.Errorf("invalid password")
	}

	if err := passStorage.Set(password); err != nil {
		return fmt.Errorf("storing password: %w", err)
	}

	fmt.Println()
	fmt.Println(ui.SuccessStyle.Render("✅ Linked successfully! Welcome back."))

	return nil
}

func createGitHubRepo(repoName string) error {
	cmd := exec.Command("gh", "repo", "create", repoName, "--private", "--description", "Vaulty secrets repository", "--confirm")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", string(output))
	}
	return nil
}

func init() {
	rootCmd.AddCommand(initCmd)
}
