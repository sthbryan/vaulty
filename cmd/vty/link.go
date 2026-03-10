package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/DeadBryam/vaulty/internal/config"
	"github.com/DeadBryam/vaulty/internal/crypto"
	"github.com/DeadBryam/vaulty/internal/github"
	"github.com/DeadBryam/vaulty/internal/ui"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

var linkCmd = &cobra.Command{
	Use:   "link",
	Short: "Link to an existing Vaulty vault on GitHub",
	Long: `Link your machine to an existing Vaulty vault on GitHub.

This command will:
  • Fetch the vault metadata from the specified GitHub repository
  • Store the configuration locally
  • Prepare you to login with 'vty login'`,
	RunE: runLink,
}

func runLink(cmd *cobra.Command, args []string) error {
	fmt.Println()
	ui.PrintAnimatedLogo()
	fmt.Println()

	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	var repoFull string

	defaultRepo := ""
	if cfg.Repo != "" {
		defaultRepo = cfg.Repo
	}

	err = huh.NewInput().
		Title("GitHub Repository").
		Placeholder(defaultRepo).
		Value(&repoFull).
		Validate(func(s string) error {
			if s == "" {
				if defaultRepo != "" {
					repoFull = defaultRepo
					return nil
				}
				return fmt.Errorf("GitHub repository is required (format: owner/repo)")
			}
			return nil
		}).
		Run()
	if err != nil {
		return fmt.Errorf("form cancelled")
	}

	if repoFull == "" {
		repoFull = defaultRepo
	}

	token, err := github.GetGitHubToken()
	if err != nil {
		return fmt.Errorf("GitHub authentication: %w", err)
	}

	client := github.NewClient(token)
	owner, repo, err := github.ParseRepo(repoFull)
	if err != nil {
		return fmt.Errorf("invalid repository format: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println()
	fmt.Println(ui.MutedStyle.Render("Fetching vault metadata..."))

	metadataResp, err := client.GetContent(ctx, owner, repo, ".vaulty/metadata.vty")
	if err != nil {
		return fmt.Errorf("fetching vault metadata: %w", err)
	}

	metadataEncData, err := client.DecodeContent(metadataResp)
	if err != nil {
		return fmt.Errorf("decoding metadata: %w", err)
	}

	metadataJSON, err := crypto.DecompressHex(string(metadataEncData))
	if err != nil {
		return fmt.Errorf("decompressing metadata: %w", err)
	}

	var metadata config.Metadata
	if err := json.Unmarshal(metadataJSON, &metadata); err != nil {
		return fmt.Errorf("parsing metadata: %w", err)
	}

	cfg.SetRepo(repoFull)
	cfg.Metadata = &metadata

	if err := cfg.Save(""); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Println()
	fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("✅ Linked to %s", repoFull)))
	fmt.Println()
	fmt.Println(ui.MutedStyle.Render("Found users:"))
	for _, user := range metadata.Users {
		fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("  • %s (%s)", user.Username, user.Role)))
	}
	fmt.Println()
	fmt.Println(ui.MutedStyle.Render("Run 'vty login' to authenticate"))

	return nil
}

func init() {
	rootCmd.AddCommand(linkCmd)
}
