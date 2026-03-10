package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/DeadBryam/vaulty/internal/cache"
	"github.com/DeadBryam/vaulty/internal/config"
	"github.com/DeadBryam/vaulty/internal/crypto"
	"github.com/DeadBryam/vaulty/internal/github"
	"github.com/DeadBryam/vaulty/internal/password"
	"github.com/DeadBryam/vaulty/internal/session"
	"github.com/DeadBryam/vaulty/internal/ui"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
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

	if cfg.Repo == "" {
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

	token, err := github.GetGitHubToken()
	if err != nil {
		return fmt.Errorf("GitHub authentication: %w", err)
	}

	client := github.NewClient(token)
	owner, repo, err := github.ParseRepo(cfg.Repo)
	if err != nil {
		return fmt.Errorf("invalid repository format: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println(ui.MutedStyle.Render("Validating credentials..."))
	metadataResp, err := client.GetContent(ctx, owner, repo, ".vaulty/metadata.vty")
	if err != nil {
		return fmt.Errorf("downloading metadata: %w", err)
	}

	metadataEncData, err := client.DecodeContent(metadataResp)
	if err != nil {
		return fmt.Errorf("decoding metadata: %w", err)
	}

	var metadata config.Metadata
	if err := json.Unmarshal(metadataEncData, &metadata); err != nil {
		return fmt.Errorf("parsing metadata: %w", err)
	}

	var userEntry *config.UserEntry
	for i := range metadata.Users {
		if metadata.Users[i].Username == username {
			userEntry = &metadata.Users[i]
			break
		}
	}

	if userEntry == nil {
		fmt.Println()
		fmt.Println(ui.ErrorStyle.Render("❌ User not found in vault"))
		fmt.Println()
		return fmt.Errorf("user %q not found in vault metadata", username)
	}

	if userEntry.PasswordChallenge != nil {
		if !crypto.ValidatePasswordWithChallenge(masterPassword, userEntry.PasswordChallenge.Salt, userEntry.PasswordChallenge.Challenge) {
			fmt.Println()
			fmt.Println(ui.ErrorStyle.Render("❌ Incorrect password"))
			fmt.Println()
			return fmt.Errorf("password validation failed")
		}
	}

	fmt.Println(ui.MutedStyle.Render("Downloading encrypted keys..."))
	keyPath := fmt.Sprintf(".vaulty/keys/%s.vty", username)
	keyResp, err := client.GetContent(ctx, owner, repo, keyPath)
	if err != nil {
		return fmt.Errorf("downloading user keys: %w", err)
	}

	keyData, err := client.DecodeContent(keyResp)
	if err != nil {
		return fmt.Errorf("decoding key data: %w", err)
	}

	fmt.Println(ui.MutedStyle.Render("Decrypting master key..."))
	keyJSON, err := crypto.DecompressHex(string(keyData))
	if err != nil {
		fmt.Println()
		fmt.Println(ui.ErrorStyle.Render("❌ Failed to decompress master key"))
		fmt.Println()
		return fmt.Errorf("decompression failed: %w", err)
	}

	encryptedKey := &crypto.EncryptedData{}
	if err := json.Unmarshal(keyJSON, encryptedKey); err != nil {
		return fmt.Errorf("parsing master key JSON: %w", err)
	}

	masterKey, err := crypto.DecryptMasterKeyWithPassword(encryptedKey, masterPassword)
	if err != nil {
		fmt.Println()
		fmt.Println(ui.ErrorStyle.Render("❌ Failed to decrypt master key"))
		fmt.Println()
		fmt.Println(ui.MutedStyle.Render("This could mean:"))
		fmt.Println(ui.MutedStyle.Render("  • Vault data is corrupted"))
		fmt.Println(ui.MutedStyle.Render("  • Try again"))
		fmt.Println()
		return fmt.Errorf("decryption failed")
	}

	fmt.Println(ui.MutedStyle.Render("Downloading vault..."))
	vaultResp, err := client.GetContent(ctx, owner, repo, ".vaulty/vault.vty")
	if err != nil {
		return fmt.Errorf("downloading vault: %w", err)
	}

	vaultEncData, err := client.DecodeContent(vaultResp)
	if err != nil {
		return fmt.Errorf("decoding vault data: %w", err)
	}

	fmt.Println(ui.MutedStyle.Render("Decrypting vault..."))
	vaultJSON, err := crypto.DecompressHex(string(vaultEncData))
	if err != nil {
		fmt.Println()
		fmt.Println(ui.ErrorStyle.Render("❌ Failed to decompress vault"))
		fmt.Println()
		return fmt.Errorf("vault decompression failed: %w", err)
	}

	encryptedVault := &crypto.EncryptedData{}
	if err := json.Unmarshal(vaultJSON, encryptedVault); err != nil {
		return fmt.Errorf("parsing vault JSON: %w", err)
	}

	vaultData, err := crypto.DecryptVaultData(encryptedVault, masterKey)
	if err != nil {
		fmt.Println()
		fmt.Println(ui.ErrorStyle.Render("❌ Failed to decrypt vault"))
		fmt.Println()
		return fmt.Errorf("vault decryption failed")
	}

	fmt.Println(ui.MutedStyle.Render("Creating session..."))
	sess := session.NewSession(username, userEntry.Role, masterKey, vaultData)
	session.GetManager().Create(sess)

	cfg.SetCurrentUser(username, userEntry.Role)
	if cfg.Metadata == nil {
		cfg.Metadata = &metadata
	} else {
		cfg.Metadata.Users = metadata.Users
	}

	if err := cfg.Save(""); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	passStorage, err := password.NewStorage()
	if err != nil {
		return fmt.Errorf("password storage: %w", err)
	}

	if err := passStorage.Set(masterPassword); err != nil {
		logger.Warn("failed to save password for auto-reauthentication", "error", err)
	}

	cacheManager := cache.NewCacheManager(passStorage)
	if err := cacheManager.Save(username, vaultData); err != nil {
		logger.Warn("failed to cache vault data", "error", err)

	}

	fmt.Println()
	fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("✅ Logged in as %s (%s)", username, userEntry.Role)))

	return nil
}

func init() {
	rootCmd.AddCommand(loginCmd)
}
