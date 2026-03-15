package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/DeadBryam/vaulty/internal/config"
	"github.com/DeadBryam/vaulty/internal/crypto"
	"github.com/DeadBryam/vaulty/internal/github"
	"github.com/DeadBryam/vaulty/internal/password"
	"github.com/DeadBryam/vaulty/internal/storage"
	"github.com/DeadBryam/vaulty/internal/ui"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

var (
	recoverUsername string
	recoverSeed     string
	recoverFile     string
)

var recoverCmd = &cobra.Command{
	Use:   "recover",
	Short: "Recover vault access using recovery seed",
	Long: `Recover your Vaulty vault access using the 12-word recovery seed phrase.

This command allows you to reset your password if you've forgotten it.
You need your username and the recovery seed provided when you were added to the vault.

Examples:
  vty recover --user john --seed "word1 word2 ... word12"
  vty recover --user john --file ~/recovery-seed.txt`,
	RunE: runRecover,
}

func runRecoverLocal(cmd *cobra.Command, args []string, cfg *config.Config, seedPhrase, username string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println()
	fmt.Println(ui.TitleStyle.Render("🔐 Set a new master password"))
	fmt.Println()

	var password1, password2 string

	err := huh.NewInput().
		Title("New master password").
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

	s, err := storage.NewLocalStorage()
	if err != nil {
		return fmt.Errorf("local storage: %w", err)
	}

	recoveryData, err := s.GetRecoverySeed(ctx, username)
	if err != nil {
		return fmt.Errorf("recovery data not found for user %s", username)
	}

	decodedRecovery, err := base64.StdEncoding.DecodeString(string(recoveryData))
	if err != nil {
		return fmt.Errorf("decoding recovery data: %w", err)
	}

	var encryptedData crypto.EncryptedData
	if err := json.Unmarshal(decodedRecovery, &encryptedData); err != nil {
		return fmt.Errorf("parsing recovery data: %w", err)
	}

	recoveredSeed, err := crypto.DecryptRecoverySeed(&encryptedData, seedPhrase)
	if err != nil {
		return fmt.Errorf("recovery seed does not match - please verify you have the correct seed for user %s", username)
	}

	if recoveredSeed != seedPhrase {
		return fmt.Errorf("recovery seed does not match")
	}

	masterKey, err := crypto.GenerateMasterKey()
	if err != nil {
		return fmt.Errorf("generating master key: %w", err)
	}

	encryptedMasterKey, err := crypto.EncryptMasterKeyWithPassword(masterKey, password1)
	if err != nil {
		return fmt.Errorf("encrypting master key: %w", err)
	}

	masterKeyJSON, err := json.Marshal(encryptedMasterKey)
	if err != nil {
		return fmt.Errorf("marshaling master key: %w", err)
	}

	err = s.PutUserKeys(ctx, username, masterKeyJSON)
	if err != nil {
		return fmt.Errorf("storing user keys: %w", err)
	}

	newEncryptedSeed, err := crypto.EncryptRecoverySeed(recoveredSeed, password1)
	if err != nil {
		return fmt.Errorf("encrypting recovery seed: %w", err)
	}

	recoverySeedJSON, err := json.Marshal(newEncryptedSeed)
	if err != nil {
		return fmt.Errorf("marshaling recovery seed: %w", err)
	}

	recoverySeedHex, err := crypto.CompressHex(recoverySeedJSON)
	if err != nil {
		return fmt.Errorf("compressing recovery seed: %w", err)
	}

	recoverySeedContent := base64.StdEncoding.EncodeToString([]byte(recoverySeedHex))
	err = s.PutRecoverySeed(ctx, username, []byte(recoverySeedContent))
	if err != nil {
		return fmt.Errorf("storing recovery seed: %w", err)
	}

	cfg.CurrentUser = username
	cfg.CurrentUserRole = "owner"

	if err := cfg.Save(""); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	passStorage, err := password.NewStorage()
	if err != nil {
		return fmt.Errorf("password storage: %w", err)
	}

	if err := passStorage.Set(password1); err != nil {
		return fmt.Errorf("storing password: %w", err)
	}

	fmt.Println()
	fmt.Println(ui.SuccessStyle.Render("✅ Recovery successful!"))
	fmt.Println()
	fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("Your vault has been recovered with a new password for user: %s", username)))
	fmt.Println(ui.MutedStyle.Render("Your vault is now ready to use on this machine."))
	fmt.Println()

	return nil
}

func runRecover(cmd *cobra.Command, args []string) error {
	var seedPhrase string

	if recoverFile != "" {
		data, err := os.ReadFile(recoverFile)
		if err != nil {
			return fmt.Errorf("reading seed file: %w", err)
		}
		seedPhrase = strings.TrimSpace(string(data))
	} else if recoverSeed != "" {
		seedPhrase = recoverSeed
	} else {
		return fmt.Errorf("--seed or --file flag is required")
	}

	if recoverUsername == "" {
		return fmt.Errorf("--user flag is required")
	}

	seedPhrase = strings.TrimSpace(strings.ToLower(seedPhrase))
	words := strings.Fields(seedPhrase)
	if len(words) != 12 {
		return fmt.Errorf("invalid recovery seed: expected 12 words, got %d", len(words))
	}

	_, err := crypto.ValidateRecoverySeed(seedPhrase)
	if err != nil {
		return fmt.Errorf("invalid recovery seed: %w", err)
	}

	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if cfg.IsLocalMode() {
		return runRecoverLocal(cmd, args, cfg, seedPhrase, recoverUsername)
	}

	if cfg.Repo == "" {
		return fmt.Errorf("no vault configured. Run 'vty init' first")
	}

	owner, repo, err := github.ParseRepo(cfg.Repo)
	if err != nil {
		return fmt.Errorf("invalid repo in config: %w", err)
	}

	token, err := github.GetGitHubToken()
	if err != nil {
		return fmt.Errorf("GitHub authentication: %w", err)
	}

	client := github.NewClient(token)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println()
	fmt.Println(ui.TitleStyle.Render("🔐 Set a new master password"))
	fmt.Println()

	var password1, password2 string

	err = huh.NewInput().
		Title("New master password").
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

	recoveryPath := fmt.Sprintf(".vaulty/recovery/%s.recovery.vty", recoverUsername)
	recoveryResp, err := client.GetContent(ctx, owner, repo, recoveryPath)
	if err != nil {
		return fmt.Errorf("recovery data not found for user %s", recoverUsername)
	}

	recoveryData, err := client.DecodeContent(recoveryResp)
	if err != nil {
		return fmt.Errorf("decoding recovery data: %w", err)
	}

	decodedRecovery, err := base64.StdEncoding.DecodeString(string(recoveryData))
	if err != nil {
		return fmt.Errorf("decoding recovery data: %w", err)
	}

	var encryptedData crypto.EncryptedData
	if err := json.Unmarshal(decodedRecovery, &encryptedData); err != nil {
		return fmt.Errorf("parsing recovery data: %w", err)
	}

	recoveredSeed, err := crypto.DecryptRecoverySeed(&encryptedData, seedPhrase)
	if err != nil {
		return fmt.Errorf("recovery seed does not match - please verify you have the correct seed for user %s", recoverUsername)
	}

	if recoveredSeed != seedPhrase {
		return fmt.Errorf("recovery seed does not match")
	}

	masterKey, err := crypto.GenerateMasterKey()
	if err != nil {
		return fmt.Errorf("generating master key: %w", err)
	}

	encryptedMasterKey, err := crypto.EncryptMasterKeyWithPassword(masterKey, password1)
	if err != nil {
		return fmt.Errorf("encrypting master key: %w", err)
	}

	masterKeyJSON, err := json.Marshal(encryptedMasterKey)
	if err != nil {
		return fmt.Errorf("marshaling master key: %w", err)
	}

	masterKeyBase64 := base64.StdEncoding.EncodeToString(masterKeyJSON)
	masterKeyPath := fmt.Sprintf(".vaulty/keys/%s.key.vty", recoverUsername)

	existingKey, err := client.GetContent(ctx, owner, repo, masterKeyPath)
	if err == nil && existingKey != nil {
		masterKeyPath = fmt.Sprintf(".vaulty/keys/%s.key.vty", recoverUsername)
	}

	err = client.PutContent(ctx, owner, repo, masterKeyPath, github.ContentRequest{
		Message: fmt.Sprintf("Update master key for %s via recovery", recoverUsername),
		Content: masterKeyBase64,
		Sha:     existingKey.Sha,
	})
	if err != nil {
		return fmt.Errorf("uploading master key: %w", err)
	}

	newEncryptedSeed, err := crypto.EncryptRecoverySeed(recoveredSeed, password1)
	if err != nil {
		return fmt.Errorf("encrypting recovery seed: %w", err)
	}

	recoverySeedJSON, err := json.Marshal(newEncryptedSeed)
	if err != nil {
		return fmt.Errorf("marshaling recovery seed: %w", err)
	}

	recoverySeedHex, err := crypto.CompressHex(recoverySeedJSON)
	if err != nil {
		return fmt.Errorf("compressing recovery seed: %w", err)
	}

	recoverySeedContent := base64.StdEncoding.EncodeToString([]byte(recoverySeedHex))
	err = client.PutContent(ctx, owner, repo, recoveryPath, github.ContentRequest{
		Message: fmt.Sprintf("Update recovery seed for %s via recovery", recoverUsername),
		Content: recoverySeedContent,
		Sha:     recoveryResp.Sha,
	})
	if err != nil {
		return fmt.Errorf("uploading recovery seed: %w", err)
	}

	cfg.CurrentUser = recoverUsername
	cfg.CurrentUserRole = "owner"

	if err := cfg.Save(""); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	passStorage, err := password.NewStorage()
	if err != nil {
		return fmt.Errorf("password storage: %w", err)
	}

	if err := passStorage.Set(password1); err != nil {
		return fmt.Errorf("storing password: %w", err)
	}

	fmt.Println()
	fmt.Println(ui.SuccessStyle.Render("✅ Recovery successful!"))
	fmt.Println()
	fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("Your vault has been recovered with a new password for user: %s", recoverUsername)))
	fmt.Println(ui.MutedStyle.Render("Your vault is now ready to use on this machine."))
	fmt.Println()

	return nil
}

func init() {
	recoverCmd.Flags().StringVar(&recoverUsername, "user", "", "Username to recover")
	recoverCmd.Flags().StringVar(&recoverSeed, "seed", "", "12-word recovery seed phrase")
	recoverCmd.Flags().StringVar(&recoverFile, "file", "", "Path to file containing the recovery seed phrase")
	rootCmd.AddCommand(recoverCmd)
}
