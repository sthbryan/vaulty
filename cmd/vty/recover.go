package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/DeadBryam/vaulty/internal/config"
	"github.com/DeadBryam/vaulty/internal/crypto"
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

func resolveUserRoleFromMetadata(ctx context.Context, s storage.Storage, username string) (string, error) {
	metadataBytes, err := s.GetMetadata(ctx)
	if err != nil {
		return "", fmt.Errorf("downloading metadata: %w", err)
	}

	var metadata config.Metadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return "", fmt.Errorf("parsing metadata: %w", err)
	}

	for _, user := range metadata.Users {
		if user.Username == username {
			if user.Role == "" {
				return "", fmt.Errorf("user %q has no role in metadata", username)
			}
			return user.Role, nil
		}
	}

	return "", fmt.Errorf("user %q not found in vault metadata", username)
}

func completeRecovery(ctx context.Context, cfg *config.Config, username, seedPhrase, newPassword string) error {
	factory := storage.NewFactory(cfg)
	s, err := factory.CreateStorage()
	if err != nil {
		return fmt.Errorf("creating storage: %w", err)
	}

	recoveryData, err := s.GetRecoverySeed(ctx, username)
	if err != nil {
		return fmt.Errorf("recovery data not found for user %s", username)
	}

	recoveryJSON, err := crypto.DecompressHex(string(recoveryData))
	if err != nil {
		return fmt.Errorf("decompressing recovery data: %w", err)
	}

	var encryptedData crypto.EncryptedData
	if err := json.Unmarshal(recoveryJSON, &encryptedData); err != nil {
		return fmt.Errorf("parsing recovery data: %w", err)
	}

	recoveredSeed, err := crypto.DecryptRecoverySeed(&encryptedData, seedPhrase)
	if err != nil {
		return fmt.Errorf("recovery seed does not match - please verify you have the correct seed for user %s", username)
	}

	if recoveredSeed != seedPhrase {
		return fmt.Errorf("recovery seed does not match")
	}

	userKeyData, err := s.GetUserKeys(ctx, username)
	if err != nil {
		return fmt.Errorf("fetching user key envelope: %w", err)
	}

	userKeyJSON, err := crypto.DecompressHex(string(userKeyData))
	if err != nil {
		return fmt.Errorf("decompressing user key envelope: %w", err)
	}

	var currentEncryptedMasterKey crypto.EncryptedData
	if err := json.Unmarshal(userKeyJSON, &currentEncryptedMasterKey); err != nil {
		return fmt.Errorf("parsing user key envelope: %w", err)
	}

	masterKey, err := crypto.DecryptMasterKeyWithPassword(&currentEncryptedMasterKey, recoveredSeed)
	if err != nil {
		return fmt.Errorf("decrypting existing user key with recovery seed: %w", err)
	}

	encryptedMasterKey, err := crypto.EncryptMasterKeyWithPassword(masterKey, newPassword)
	if err != nil {
		return fmt.Errorf("encrypting master key: %w", err)
	}

	masterKeyJSON, err := json.Marshal(encryptedMasterKey)
	if err != nil {
		return fmt.Errorf("marshaling master key: %w", err)
	}

	masterKeyHex, err := crypto.CompressHex(masterKeyJSON)
	if err != nil {
		return fmt.Errorf("compressing master key: %w", err)
	}

	if err := s.PutUserKeys(ctx, username, []byte(masterKeyHex)); err != nil {
		return fmt.Errorf("storing user keys: %w", err)
	}

	newEncryptedSeed, err := crypto.EncryptRecoverySeed(recoveredSeed, newPassword)
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

	if err := s.PutRecoverySeed(ctx, username, []byte(recoverySeedHex)); err != nil {
		return fmt.Errorf("storing recovery seed: %w", err)
	}

	role, err := resolveUserRoleFromMetadata(ctx, s, username)
	if err != nil {
		return err
	}

	cfg.CurrentUser = username
	cfg.CurrentUserRole = role

	if err := cfg.Save(""); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	passStorage, err := password.NewStorage()
	if err != nil {
		return fmt.Errorf("password storage: %w", err)
	}

	if err := passStorage.Set(newPassword); err != nil {
		return fmt.Errorf("storing password: %w", err)
	}

	return nil
}

func runRecoverLocal(cfg *config.Config, seedPhrase, username string) error {
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

	if err := completeRecovery(ctx, cfg, username, seedPhrase, password1); err != nil {
		return err
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
		return runRecoverLocal(cfg, seedPhrase, recoverUsername)
	}

	if cfg.Repo == "" {
		return fmt.Errorf("no vault configured. Run 'vty init' first")
	}

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

	if err := completeRecovery(ctx, cfg, recoverUsername, seedPhrase, password1); err != nil {
		return err
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
