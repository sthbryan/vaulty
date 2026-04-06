package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/sthbryan/vaulty/internal/cli"
	"github.com/sthbryan/vaulty/internal/compress"
	"github.com/sthbryan/vaulty/internal/crypto"
	"github.com/sthbryan/vaulty/internal/ui"
	"github.com/sthbryan/vaulty/pkg/models"
)

type resourceVaultFile struct {
	Metadata models.ResourceMetadata `json:"metadata"`
	Data     []byte                  `json:"data"`
}

var pullResourceTag string

var pullResourceCmd = &cobra.Command{
	Use:   "resource <name>",
	Short: "Pull a file or directory from resources",
	Long: `Download and decrypt a file or directory from the resources/ directory.

Examples:
  vty pull resource agents
  vty pull resource zellij --tag dev
  vty pull resource config.yml --tag team -o ./config.yml`,
	Args: cobra.ExactArgs(1),
	RunE: runPullResource,
}

var pullConfigCmd = &cobra.Command{
	Use:   "config <name>",
	Short: "Pull a file or directory from config",
	Long: `Download and decrypt a file or directory from the config/ directory.

Examples:
  vty pull config opencode
  vty pull config zellij --tag team
  vty pull config settings --tag dev -o ./settings`,
	Args: cobra.ExactArgs(1),
	RunE: runPullConfig,
}

func runPullResource(cmd *cobra.Command, args []string) error {
	return runPullResourceOrConfig(args[0], "resources")
}

func runPullConfig(cmd *cobra.Command, args []string) error {
	return runPullResourceOrConfig(args[0], "config")
}

func runPullResourceOrConfig(name, baseDir string) error {
	if err := validateName(name); err != nil {
		return err
	}
	if pullResourceTag != "" {
		if err := validateName(pullResourceTag); err != nil {
			return fmt.Errorf("invalid tag: %w", err)
		}
	}

	cfg, s, err := loadConfigAndStorage()
	if err != nil {
		return err
	}

	var _ = s

	sess, err := ensureAuthenticated(cfg)
	if err != nil {
		return err
	}

	var remotePath string
	if pullResourceTag != "" {
		remotePath = fmt.Sprintf("%s/%s/%s.vty", baseDir, pullResourceTag, name)
	} else {
		remotePath = fmt.Sprintf("%s/%s.vty", baseDir, name)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logger.Info("☁️  Downloading resource...", "name", name)

	data, err := s.GetResource(ctx, remotePath)
	if err != nil {
		return fmt.Errorf("resource not found: %s (try with --tag flag)", remotePath)
	}

	logger.Info("✓ Downloaded", "path", remotePath, "size", len(data))

	logger.Info("🔓 Decrypting...")

	vaultJSON, err := crypto.DecryptBinary(string(data), sess.MasterKey)
	if err != nil {
		if err == crypto.ErrDecryptionFailed {
			return fmt.Errorf("decryption failed: invalid password")
		}
		return fmt.Errorf("decrypting: %w", err)
	}

	var vaultFile resourceVaultFile
	if err := json.Unmarshal(vaultJSON, &vaultFile); err != nil {
		return fmt.Errorf("parsing vault file: %w", err)
	}

	isDirectory := vaultFile.Metadata.IsDirectory

	var plaintext []byte
	if isDirectory {
		ui.PrintInfo("Decompressing directory...")
		plaintext = vaultFile.Data
	} else {
		plaintext = vaultFile.Data
	}

	outputFile, err := cli.ResolveOutputPath(pullOutput, name)
	if err != nil {
		return err
	}

	if isDirectory {
		targetDir := outputFile
		if pullOutput == "" {
			targetDir = filepath.Join(filepath.Dir(outputFile), name)
		}

		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return fmt.Errorf("creating directory: %w", err)
		}
		if err := compress.DecompressDirectory(plaintext, targetDir); err != nil {
			return fmt.Errorf("decompressing directory: %w", err)
		}
		logger.Info("💾 Saved", "path", targetDir, "type", "directory")
		fmt.Println()
		fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("✅ Pulled directory: %s", targetDir)))
	} else {
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

		parentDir := filepath.Dir(outputFile)
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			return fmt.Errorf("creating parent directory: %w", err)
		}

		if err := os.WriteFile(outputFile, plaintext, 0600); err != nil {
			return fmt.Errorf("writing file: %w", err)
		}

		logger.Info("💾 Saved", "path", outputFile, "size", len(plaintext))
		fmt.Println()
		fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("✅ Pulled: %s", outputFile)))
	}

	fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("   Directory: %v", isDirectory)))

	return nil
}

func init() {
	pullResourceCmd.Flags().StringVarP(&pullResourceTag, "tag", "t", "", "Tag of the resource (e.g., dev, team)")
	pullResourceCmd.Flags().StringVarP(&pullOutput, "output", "o", "", "Output filename or directory")
	pullResourceCmd.Flags().BoolVarP(&pullInteractive, "interactive", "i", false, "Interactive mode")

	pullConfigCmd.Flags().StringVarP(&pullResourceTag, "tag", "t", "", "Tag of the config (e.g., dev, team)")
	pullConfigCmd.Flags().StringVarP(&pullOutput, "output", "o", "", "Output filename or directory")
	pullConfigCmd.Flags().BoolVarP(&pullInteractive, "interactive", "i", false, "Interactive mode")

	pullCmd.AddCommand(pullResourceCmd)
	pullCmd.AddCommand(pullConfigCmd)
}
