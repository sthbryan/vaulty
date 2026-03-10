package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/DeadBryam/vaulty/internal/config"
	"github.com/DeadBryam/vaulty/internal/crypto"
	"github.com/DeadBryam/vaulty/internal/github"
	"github.com/DeadBryam/vaulty/internal/ui"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

var addUserCmd = &cobra.Command{
	Use:   "add-user <username>",
	Short: "Add a new user to the vault",
	Long: `Add a new user to your Vaulty vault.

This command allows the vault owner to add new users with viewer access.
You must provide your master password to verify ownership.`,
	Args: cobra.ExactArgs(1),
	RunE: runAddUser,
}

func runAddUser(cmd *cobra.Command, args []string) error {
	username := args[0]

	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if cfg.Repo == "" {
		return fmt.Errorf("no vault initialized - run 'vty init' first")
	}

	if cfg.CurrentUserRole != "owner" {
		return fmt.Errorf("only vault owner can add users")
	}

	owner, repo, err := github.ParseRepo(cfg.Repo)
	if err != nil {
		return fmt.Errorf("parsing repo: %w", err)
	}

	token, err := github.GetGitHubToken()
	if err != nil {
		return fmt.Errorf("GitHub authentication: %w", err)
	}

	client := github.NewClient(token)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

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
	fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Downloading .vaulty/keys/%s.enc...", cfg.CurrentUser)))

	keyPath := fmt.Sprintf(".vaulty/keys/%s.enc", cfg.CurrentUser)
	keyResp, err := client.GetContent(ctx, owner, repo, keyPath)
	if err != nil {
		return fmt.Errorf("failed to download owner key: %w", err)
	}

	keyData, err := client.DecodeContent(keyResp)
	if err != nil {
		return fmt.Errorf("decoding owner key: %w", err)
	}

	encryptedData, err := crypto.DeserializeEncryptedData(keyData)
	if err != nil {
		return fmt.Errorf("deserializing owner key: %w", err)
	}

	masterKey, err := crypto.DecryptMasterKeyWithPassword(encryptedData, ownerPassword)
	if err != nil {
		fmt.Println()
		fmt.Println(ui.ErrorStyle.Render("❌ Failed to decrypt vault"))
		fmt.Println()
		fmt.Println(ui.MutedStyle.Render("Invalid password or corrupted vault data."))
		fmt.Println()
		return fmt.Errorf("decryption failed")
	}

	fmt.Println()
	fmt.Println(ui.MutedStyle.Render("Checking metadata..."))

	metadataBytes, err := client.GetMetadata(ctx, owner, repo)
	if err != nil {
		return fmt.Errorf("failed to download metadata: %w", err)
	}

	var metadata config.Metadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return fmt.Errorf("parsing metadata: %w", err)
	}

	for _, user := range metadata.Users {
		if user.Username == username {
			return fmt.Errorf("user %q already exists", username)
		}
	}

	fmt.Println()
	fmt.Println(ui.InfoStyle.Render("🔑 Create new user password"))
	fmt.Println()

	var newPassword1, newPassword2 string

	err = huh.NewInput().
		Title("New password").
		Placeholder("Enter a strong password").
		EchoMode(huh.EchoModePassword).
		Value(&newPassword1).
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
		Placeholder("Re-enter password").
		EchoMode(huh.EchoModePassword).
		Value(&newPassword2).
		Validate(func(s string) error {
			if s != newPassword1 {
				return fmt.Errorf("passwords do not match")
			}
			return nil
		}).
		Run()
	if err != nil {
		return fmt.Errorf("form cancelled")
	}

	fmt.Println()
	fmt.Println(ui.MutedStyle.Render("Encrypting master key..."))

	encryptedMasterKey, err := crypto.EncryptMasterKeyWithPassword(masterKey, newPassword1)
	if err != nil {
		return fmt.Errorf("encrypting master key: %w", err)
	}

	masterKeyBytes := crypto.SerializeEncryptedData(encryptedMasterKey)
	masterKeyContent := base64.StdEncoding.EncodeToString(masterKeyBytes)

	recoverySeeds, err := crypto.GenerateRecoverySeed()
	if err != nil {
		return fmt.Errorf("generating recovery seed: %w", err)
	}

	metadata.Users = append(metadata.Users, config.UserEntry{
		Username:  username,
		Role:      "viewer",
		CreatedAt: time.Now(),
	})

	metadataJSON, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling metadata: %w", err)
	}

	fmt.Println()
	fmt.Println(ui.MutedStyle.Render("Uploading files to GitHub..."))

	err = client.PutContent(ctx, owner, repo, fmt.Sprintf(".vaulty/keys/%s.enc", username), github.ContentRequest{
		Message: fmt.Sprintf("Add user %s", username),
		Content: masterKeyContent,
	})
	if err != nil {
		return fmt.Errorf("uploading key: %w", err)
	}

	encryptedSeed, err := crypto.EncryptRecoverySeed(recoverySeeds, newPassword1)
	if err != nil {
		return fmt.Errorf("encrypting recovery seed: %w", err)
	}

	encryptedSeedJSON, err := json.Marshal(encryptedSeed)
	if err != nil {
		return fmt.Errorf("marshaling encrypted seed: %w", err)
	}

	recoveryContent := base64.StdEncoding.EncodeToString(encryptedSeedJSON)
	err = client.PutRecoverySeed(ctx, owner, repo, username, []byte(recoveryContent))
	if err != nil {
		return fmt.Errorf("uploading recovery seed: %w", err)
	}

	err = client.PutMetadata(ctx, owner, repo, metadataJSON)
	if err != nil {
		return fmt.Errorf("uploading metadata: %w", err)
	}

	fmt.Println()
	fmt.Println(ui.SuccessStyle.Render("✅ User created successfully!"))
	fmt.Println()
	fmt.Println(ui.WarningStyle.Render("⚠️  Recovery seed for new user:"))
	fmt.Println()
	fmt.Println(ui.TitleStyle.Render(recoverySeeds))
	fmt.Println()

	saveToFile, err := ui.AskConfirm("Save recovery seed to a file?", true)
	if err != nil {
		return fmt.Errorf("confirmation failed: %w", err)
	}

	if saveToFile {
		defaultPath := fmt.Sprintf("vaulty-recovery-%s.txt", username)
		var filePath string

		err = huh.NewInput().
			Title("File path").
			Placeholder(defaultPath).
			Value(&filePath).
			Run()
		if err != nil {
			return fmt.Errorf("form cancelled")
		}

		if filePath == "" {
			filePath = defaultPath
		}

		if err := os.WriteFile(filePath, []byte(recoverySeeds), 0600); err != nil {
			return fmt.Errorf("saving seed file: %w", err)
		}

		fmt.Println()
		fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("✅ Recovery seed saved to: %s", filePath)))
		fmt.Println(ui.MutedStyle.Render("Share this file securely with the new user."))
	}

	fmt.Println()
	fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("📝 Username: %s", username)))
	fmt.Println(ui.InfoStyle.Render("Role: viewer"))
	fmt.Println()

	return nil
}

func init() {
	rootCmd.AddCommand(addUserCmd)
}
