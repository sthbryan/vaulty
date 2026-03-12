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
	"github.com/DeadBryam/vaulty/internal/session"
	"github.com/DeadBryam/vaulty/internal/ui"
	"github.com/charmbracelet/huh"
)

func pullSecretWithRemotePath(name, remotePath string, sess *session.Session) error {
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

	token, err := github.GetGitHubToken()
	if err != nil {
		return fmt.Errorf("getting GitHub token: %w", err)
	}

	client := github.NewClient(token)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logger.Info("☁️  Downloading secrets...", "name", name)

	content, err := client.GetContent(ctx, owner, repo, remotePath)
	if err != nil {
		return fmt.Errorf("secret not found: %s", remotePath)
	}

	logger.Info("✓ Downloaded", "path", remotePath, "size", content.Size)

	encodedData, err := client.DecodeContent(content)
	if err != nil {
		return fmt.Errorf("decoding content: %w", err)
	}

	logger.Info("🔓 Decrypting...")
	hexData := string(encodedData)
	vaultJSON, err := crypto.DecryptBinary(hexData, sess.MasterKey)
	if err != nil {
		if err == crypto.ErrDecryptionFailed {
			return fmt.Errorf("decryption failed: invalid password")
		}
		return fmt.Errorf("decrypting: %w", err)
	}

	var vaultFile BinaryVaultFile
	if err := json.Unmarshal(vaultJSON, &vaultFile); err != nil {
		return fmt.Errorf("parsing vault file: %w", err)
	}

	plaintext, err := compress.Decompress(vaultFile.Data)
	if err != nil {
		return fmt.Errorf("decompressing data: %w", err)
	}

	outputFile, err := getOutputFilename(name, "env")
	if err != nil {
		return err
	}

	if outputFile == "-" {
		fmt.Print(string(plaintext))
		return nil
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
	fmt.Println(ui.MutedStyle.Render("   Permissions: 0600 (owner read/write only)"))

	return nil
}

func pullSecretWithSession(name, secretType, targetUser string, sess *session.Session) error {
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

	token, err := github.GetGitHubToken()
	if err != nil {
		return fmt.Errorf("getting GitHub token: %w", err)
	}

	client := github.NewClient(token)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var path string
	if secretType == "env" {
		path = fmt.Sprintf("envs/%s.vty", name)
		logger.Info("☁️  Downloading environment secrets...", "name", name)
	} else {
		path = fmt.Sprintf("ssh/%s/%s.vty", targetUser, name)
		logger.Info("☁️  Downloading SSH key...", "name", name, "user", targetUser)
	}

	content, err := client.GetContent(ctx, owner, repo, path)
	if err != nil {
		return fmt.Errorf("secret not found: %s", path)
	}

	logger.Info("✓ Downloaded", "path", path, "size", content.Size)

	encodedData, err := client.DecodeContent(content)
	if err != nil {
		return fmt.Errorf("decoding content: %w", err)
	}

	logger.Info("🔓 Decrypting...")
	hexData := string(encodedData)
	vaultJSON, err := crypto.DecryptBinary(hexData, sess.MasterKey)
	if err != nil {
		if err == crypto.ErrDecryptionFailed {
			return fmt.Errorf("decryption failed: invalid password")
		}
		return fmt.Errorf("decrypting: %w", err)
	}

	var vaultFile BinaryVaultFile
	if err := json.Unmarshal(vaultJSON, &vaultFile); err != nil {
		return fmt.Errorf("parsing vault file: %w", err)
	}

	plaintext, err := compress.Decompress(vaultFile.Data)
	if err != nil {
		return fmt.Errorf("decompressing data: %w", err)
	}

	outputFile, err := getOutputFilename(name, secretType)
	if err != nil {
		return err
	}

	if outputFile == "-" {
		fmt.Print(string(plaintext))
		return nil
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
	fmt.Println(ui.MutedStyle.Render("   Permissions: 0600 (owner read/write only)"))

	return nil
}

func getOutputFilename(name, secretType string) (string, error) {
	if pullOutput != "" {
		return pullOutput, nil
	}

	if !pullInteractive {
		if secretType == "ssh" {
			return name, nil
		}
		return ".env", nil
	}

	fmt.Println()
	fmt.Println(ui.InfoStyle.Render("💾 Choose output filename:"))

	var selected string
	var options []huh.Option[string]

	if secretType == "ssh" {
		options = []huh.Option[string]{
			huh.NewOption(fmt.Sprintf("%s (default)", name), name),
			huh.NewOption("Custom filename...", "custom"),
		}
	} else {
		options = []huh.Option[string]{
			huh.NewOption(".env (default)", ".env"),
			huh.NewOption(".env.local", ".env.local"),
			huh.NewOption(".env.production", ".env.production"),
			huh.NewOption(".env.development", ".env.development"),
			huh.NewOption("Custom filename...", "custom"),
		}
	}

	err := huh.NewSelect[string]().
		Title("Select filename").
		Options(options...).
		Value(&selected).
		Run()
	if err != nil {
		return "", fmt.Errorf("selection cancelled")
	}

	if selected == "custom" {
		defaultName := name
		if secretType == "env" {
			defaultName = "my-secrets.env"
		}
		customName, err := ui.AskInput("Enter custom filename", defaultName)
		if err != nil {
			return "", fmt.Errorf("input cancelled")
		}
		return customName, nil
	}

	return selected, nil
}
