package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/DeadBryam/vaulty/internal/config"
	"github.com/DeadBryam/vaulty/internal/crypto"
	"github.com/DeadBryam/vaulty/internal/github"
	"github.com/DeadBryam/vaulty/internal/password"
	"github.com/DeadBryam/vaulty/internal/ui"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

var recoverCmd = &cobra.Command{
	Use:   "recover",
	Short: "Recover vault using seed phrase",
	Long:  "Recover your Vaulty vault using the 12-word recovery seed phrase.",
	RunE:  runRecover,
}

var recoverSeed string
var recoverFile string

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

	seedPhrase = strings.TrimSpace(strings.ToLower(seedPhrase))
	words := strings.Fields(seedPhrase)
	if len(words) != 12 {
		return fmt.Errorf("invalid recovery seed phrase: expected 12 words, got %d", len(words))
	}

	_, err := crypto.ValidateRecoverySeed(seedPhrase)
	if err != nil {
		return fmt.Errorf("invalid recovery seed phrase: %w", err)
	}

	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
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

	fmt.Println(ui.InfoStyle.Render("🔐 Set a new master password"))
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

	deviceSalt := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, deviceSalt); err != nil {
		return fmt.Errorf("generating device salt: %w", err)
	}
	cfg.DeviceSalt = deviceSalt

	recoveryResp, err := client.GetContent(ctx, owner, repo, ".vaulty/recovery.vty")
	if err != nil {
		return fmt.Errorf("recovery data not found in vault - this vault may have been created before recovery support was added. Please recreate the vault with 'vty init'")
	}

	recoveryData, err := client.DecodeContent(recoveryResp)
	if err != nil {
		return fmt.Errorf("decoding recovery data: %w", err)
	}

	originalPassword, err := crypto.DecryptPasswordWithSeed(recoveryData, seedPhrase)
	if err != nil {
		return fmt.Errorf("seed phrase does not match this vault - please verify you have the correct seed phrase")
	}

	canaryResp, err := client.GetContent(ctx, owner, repo, ".vaulty/canary.vty")
	if err != nil {
		return fmt.Errorf("fetching canary: %w", err)
	}

	canaryData, err := client.DecodeContent(canaryResp)
	if err != nil {
		return fmt.Errorf("decoding canary: %w", err)
	}

	if err := crypto.ValidateCanary(canaryData, originalPassword, deviceSalt); err != nil {
		return fmt.Errorf("invalid seed phrase")
	}

	saltResp, err := client.GetContent(ctx, owner, repo, ".vaulty/salt.vty")
	if err != nil {
		return fmt.Errorf("device salt not found in vault: %w", err)
	}

	saltData, err := client.DecodeContent(saltResp)
	if err != nil {
		return fmt.Errorf("decoding device salt: %w", err)
	}

	vaultDeviceSalt, err := crypto.DecryptDeviceSalt(saltData, originalPassword)
	if err != nil {
		return fmt.Errorf("decrypting device salt: %w", err)
	}

	cfg.DeviceSalt = vaultDeviceSalt

	newSalt, err := crypto.EncryptDeviceSalt(vaultDeviceSalt, password1)
	if err != nil {
		return fmt.Errorf("encrypting device salt with new password: %w", err)
	}

	saltContent := base64.StdEncoding.EncodeToString(newSalt)
	err = client.PutContent(ctx, owner, repo, ".vaulty/salt.vty", github.ContentRequest{
		Message: "Recover vault - update salt encryption",
		Content: saltContent,
		Sha:     saltResp.Sha,
	})
	if err != nil {
		return fmt.Errorf("updating device salt: %w", err)
	}

	newCanary, err := crypto.GenerateCanary(password1, vaultDeviceSalt)
	if err != nil {
		return fmt.Errorf("generating new canary: %w", err)
	}

	canaryContent := base64.StdEncoding.EncodeToString(newCanary)
	err = client.PutContent(ctx, owner, repo, ".vaulty/canary.vty", github.ContentRequest{
		Message: "Recover vault - update canary",
		Content: canaryContent,
		Sha:     canaryResp.Sha,
	})
	if err != nil {
		return fmt.Errorf("updating canary: %w", err)
	}

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
	fmt.Println(ui.InfoStyle.Render("Your vault has been recovered with a new master password."))
	fmt.Println(ui.MutedStyle.Render("Your vault is now ready to use on this machine."))
	fmt.Println()

	return nil
}

func init() {
	recoverCmd.Flags().StringVar(&recoverSeed, "seed", "", "12-word recovery seed phrase")
	recoverCmd.Flags().StringVar(&recoverFile, "file", "", "Path to file containing the recovery seed phrase")
	rootCmd.AddCommand(recoverCmd)
}
