package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"github.com/sthbryan/vaulty/internal/compress"
	"github.com/sthbryan/vaulty/internal/config"
	"github.com/sthbryan/vaulty/internal/crypto"
	"github.com/sthbryan/vaulty/internal/github"
	"github.com/sthbryan/vaulty/internal/ui"
)

var (
	pullOutput        string
	pullInteractive   bool
	pullPassword      string
	pullPasswordStdin bool
)

var pullCmd = &cobra.Command{
	Use:   "pull <name>",
	Short: "☁️  Pull and decrypt secrets from GitHub",
	Long: `Pull encrypted secrets from your GitHub repository.

This command will:
  • Download the encrypted file from GitHub (tries envs/ then ssh/)
  • Decrypt and decompress the data 🔓🗜️
  • Save to your chosen filename with secure permissions (0600) 🔐

Examples:
  vty pull myapp-prod
  vty pull myapp-prod -o .env.production
  vty pull myapp-prod --password mypass
  echo "mypass" | vty pull myapp-prod --password-stdin`,
	Args: cobra.ExactArgs(1),
	RunE: runPull,
}

func runPull(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Load config
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	owner, repo, err := github.ParseRepo(cfg.Repo)
	if err != nil {
		return fmt.Errorf("parsing repo: %w", err)
	}

	// Get GitHub token
	token, err := github.GetGitHubToken()
	if err != nil {
		return fmt.Errorf("getting GitHub token: %w", err)
	}

	client := github.NewClient(token)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Try to download from envs/ first, then ssh/
	logger.Info("☁️  Downloading from GitHub...", "name", name)

	var content *github.ContentResponse
	var path string

	// Try envs/ first
	path = fmt.Sprintf("envs/%s.vty", name)
	content, err = client.GetContent(ctx, owner, repo, path)
	if err != nil {
		// Try ssh/ next
		logger.Info("Not found in envs/, trying ssh/...")
		path = fmt.Sprintf("ssh/%s.vty", name)
		content, err = client.GetContent(ctx, owner, repo, path)
		if err != nil {
			return fmt.Errorf("secret not found in envs/ or ssh/: %w", err)
		}
	}

	logger.Info("✓ Downloaded", "path", path, "size", content.Size)

	// Decode content
	encodedData, err := client.DecodeContent(content)
	if err != nil {
		return fmt.Errorf("decoding content: %w", err)
	}

	// Deserialize encrypted data
	encryptedData, err := crypto.DeserializeEncryptedData(encodedData)
	if err != nil {
		return fmt.Errorf("deserializing encrypted data: %w", err)
	}

	// Get password
	password, err := getPassword()
	if err != nil {
		return err
	}

	// Decrypt
	logger.Info("🔓 Decrypting...")
	compressedData, err := crypto.Decrypt(encryptedData, password)
	if err != nil {
		if err == crypto.ErrDecryptionFailed {
			return fmt.Errorf("decryption failed: invalid password")
		}
		return fmt.Errorf("decrypting: %w", err)
	}

	// Decompress
	logger.Info("🗜️  Decompressing...")
	plaintext, err := compress.Decompress(compressedData)
	if err != nil {
		return fmt.Errorf("decompressing: %w", err)
	}

	// Determine output filename
	outputFile, err := getOutputFilename(name)
	if err != nil {
		return err
	}

	// Make path absolute
	if !filepath.IsAbs(outputFile) {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting working directory: %w", err)
		}
		outputFile = filepath.Join(cwd, outputFile)
	}

	// Check if file exists
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

	// Save file with 0600 permissions
	if err := os.WriteFile(outputFile, plaintext, 0600); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	logger.Info("💾 Saved", "path", outputFile, "size", len(plaintext))
	fmt.Println()
	fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("✅ Pulled and decrypted: %s", outputFile)))
	fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("   Permissions: 0600 (owner read/write only)")))

	return nil
}

func getPassword() (string, error) {
	// Priority: --password-stdin > --password > interactive prompt

	if pullPasswordStdin {
		reader := bufio.NewReader(os.Stdin)
		password, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("reading password from stdin: %w", err)
		}
		return strings.TrimSpace(password), nil
	}

	if pullPassword != "" {
		return pullPassword, nil
	}

	// Interactive prompt
	password, err := ui.AskPassword("🔐 Enter decryption password")
	if err != nil {
		return "", fmt.Errorf("password prompt cancelled")
	}

	return password, nil
}

func getOutputFilename(name string) (string, error) {
	// If -o flag is set, use that
	if pullOutput != "" {
		return pullOutput, nil
	}

	// If not interactive, default to .env
	if !pullInteractive {
		return ".env", nil
	}

	// Interactive filename selection
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
	pullCmd.Flags().StringVar(&pullPassword, "password", "", "Decryption password")
	pullCmd.Flags().BoolVar(&pullPasswordStdin, "password-stdin", false, "Read password from stdin")

	rootCmd.AddCommand(pullCmd)
}

// SetLogger for testing
func SetLogger(l *log.Logger) {
	logger = l
}
