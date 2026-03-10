package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/DeadBryam/vaulty/internal/compress"
	"github.com/DeadBryam/vaulty/internal/config"
	"github.com/DeadBryam/vaulty/internal/crypto"
	"github.com/DeadBryam/vaulty/internal/github"
	"github.com/DeadBryam/vaulty/internal/password"
	"github.com/DeadBryam/vaulty/internal/ui"
	"github.com/DeadBryam/vaulty/pkg/models"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

var (
	pullOutput      string
	pullInteractive bool
)

var pullCmd = &cobra.Command{
	Use:   "pull <name>",
	Short: "Pull and decrypt secrets from GitHub",
	Long: `Pull encrypted secrets from your GitHub repository.

This command will:
  • Download the encrypted file from GitHub (tries envs/ then ssh/)
  • Decrypt and decompress the data 🔓🗜️
  • Save to your chosen filename with secure permissions (0600) 🔐

Examples:
  vty pull myapp-prod
  vty pull myapp-prod -o .env.production`,
	Args: cobra.ExactArgs(1),
	RunE: runPull,
}

func runPull(cmd *cobra.Command, args []string) error {
	name := args[0]

	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	if err := cfg.ValidateAndRefreshSession(); err != nil {
		return fmt.Errorf("session validation failed: %w", err)
	}

	owner, repo, err := github.ParseRepo(cfg.Repo)
	if err != nil {
		return fmt.Errorf("parsing repo: %w", err)
	}

	token, err := github.GetGitHubToken()
	if err != nil {
		return fmt.Errorf("getting GitHub token: %w", err)
	}

	client := github.NewClient(token)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logger.Info("☁️  Downloading from GitHub...", "name", name)

	storage, err := password.NewStorage()
	if err != nil {
		return fmt.Errorf("initializing password storage: %w", err)
	}

	passwordStr, err := storage.Get()
	if err != nil {
		return fmt.Errorf("Password not found. Run 'vty init' or 'vty recover'")
	}

	var content *github.ContentResponse
	var path string

	path = fmt.Sprintf("envs/%s.vty", name)
	content, err = client.GetContent(ctx, owner, repo, path)
	if err != nil {
		logger.Info("Not found in envs/, trying ssh/...")
		path = fmt.Sprintf("ssh/%s.vty", name)
		content, err = client.GetContent(ctx, owner, repo, path)
		if err != nil {
			return fmt.Errorf("secret not found in envs/ or ssh/: %w", err)
		}
	}

	logger.Info("✓ Downloaded", "path", path, "size", content.Size)

	encodedData, err := client.DecodeContent(content)
	if err != nil {
		return fmt.Errorf("decoding content: %w", err)
	}

	var vaultFile models.VaultFile
	if err := json.Unmarshal(encodedData, &vaultFile); err != nil {
		return fmt.Errorf("unmarshaling vault file: %w", err)
	}

	logger.Info("🔓 Decrypting...")
	compressedData, err := crypto.Decrypt(&vaultFile.Data, passwordStr)
	if err != nil {
		if err == crypto.ErrDecryptionFailed {
			return fmt.Errorf("decryption failed: invalid password")
		}
		return fmt.Errorf("decrypting: %w", err)
	}

	logger.Info("🗜️  Decompressing...")
	plaintext, err := compress.Decompress(compressedData)
	if err != nil {
		return fmt.Errorf("decompressing: %w", err)
	}

	outputFile, err := getOutputFilename(name)
	if err != nil {
		return err
	}

	if !filepath.IsAbs(outputFile) {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting working directory: %w", err)
		}
		outputFile = filepath.Join(cwd, outputFile)
	}

	if _, err := os.Stat(outputFile); err == nil {
		if pullInteractive {
			confirmed, err := ui.AskConfirm(fmt.Sprintf("File %s already exists. Overwrite?", outputFile), false)
			if err != nil {
				return fmt.Errorf("prompt cancelled")
			}
			if !confirmed {
				logger.Info("Aborted")
				return nil
			}
		} else {
			return fmt.Errorf("file already exists: %s (use -i to overwrite)", outputFile)
		}
	}

	if err := os.WriteFile(outputFile, plaintext, 0600); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	logger.Info("💾 Saved", "path", outputFile, "size", len(plaintext))
	fmt.Println()
	fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("✅ Pulled and decrypted: %s", outputFile)))
	fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("   Permissions: 0600 (owner read/write only)")))

	return nil
}

func getOutputFilename(name string) (string, error) {
	if pullOutput != "" {
		return pullOutput, nil
	}

	if !pullInteractive {
		return ".env", nil
	}

	fmt.Println()
	fmt.Println(ui.InfoStyle.Render("💾 Choose output filename:"))

	var selected string
	err := huh.NewSelect[string]().
		Title("Select filename").
		Options(
			huh.NewOption(".env (default)", ".env"),
			huh.NewOption(".env.local", ".env.local"),
			huh.NewOption(".env.production", ".env.production"),
			huh.NewOption(".env.development", ".env.development"),
			huh.NewOption("Custom filename...", "custom"),
		).
		Value(&selected).
		Run()
	if err != nil {
		return "", fmt.Errorf("selection cancelled")
	}

	if selected == "custom" {
		customName, err := ui.AskInput("Enter custom filename", "my-secrets.env")
		if err != nil {
			return "", fmt.Errorf("input cancelled")
		}
		return customName, nil
	}

	return selected, nil
}

func init() {
	pullCmd.Flags().StringVarP(&pullOutput, "output", "o", "", "Output filename (default: .env)")
	pullCmd.Flags().BoolVarP(&pullInteractive, "interactive", "i", false, "Interactive mode (prompt for filename)")

	rootCmd.AddCommand(pullCmd)
}

func SetLogger(l *log.Logger) {
	logger = l
}
