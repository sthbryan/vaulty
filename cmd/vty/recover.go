package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"github.com/sthbryan/vaulty/internal/config"
	"github.com/sthbryan/vaulty/internal/crypto"
	"github.com/sthbryan/vaulty/internal/github"
	"github.com/sthbryan/vaulty/internal/password"
	"github.com/sthbryan/vaulty/internal/ui"
)

var recoverCmd = &cobra.Command{
	Use:   "recover",
	Short: "Recover vault using seed phrase",
	Long:  "Recover your Vaulty vault using the 12-word recovery seed phrase.",
	RunE:  runRecover,
}

var recoverSeed string

func runRecover(cmd *cobra.Command, args []string) error {
	if recoverSeed == "" {
		return fmt.Errorf("--seed flag is required")
	}

	words := strings.Fields(recoverSeed)
	if len(words) != 12 {
		return fmt.Errorf("Invalid recovery seed phrase")
	}

	_, err := crypto.ValidateRecoverySeed(recoverSeed)
	if err != nil {
		return fmt.Errorf("Invalid recovery seed phrase")
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

	canaryResp, err := client.GetContent(ctx, owner, repo, ".vaulty/canary.vty")
	if err != nil {
		return fmt.Errorf("fetching canary: %w", err)
	}

	canaryData, err := client.DecodeContent(canaryResp)
	if err != nil {
		return fmt.Errorf("decoding canary: %w", err)
	}

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

	encrypted, err := crypto.DeserializeEncryptedData(canaryData)
	if err != nil {
		return fmt.Errorf("Recovery failed: seed does not match this vault")
	}

	_, err = crypto.Decrypt(encrypted, password1+string(deviceSalt))
	if err != nil {
		return fmt.Errorf("Recovery failed: seed does not match this vault")
	}

	newCanary, err := crypto.GenerateCanary(password1, deviceSalt)
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
	fmt.Println(ui.MutedStyle.Render("The device salt has been regenerated for this machine."))
	fmt.Println()

	return nil
}

func init() {
	recoverCmd.Flags().StringVar(&recoverSeed, "seed", "", "12-word recovery seed phrase")
	recoverCmd.MarkFlagRequired("seed")
}
