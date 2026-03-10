package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/DeadBryam/vaulty/internal/config"
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
	cfg, err := config.Load("")
	if err != nil {
		cfg = &config.Config{}
	}

	if cfg.Repo != "" {
		alreadyLinked, err := ui.AskConfirm(fmt.Sprintf("Already linked to %s. Replace with new vault?", cfg.Repo), false)
		if err != nil {
			return fmt.Errorf("confirmation failed: %w", err)
		}
		if !alreadyLinked {
			fmt.Println("Link cancelled")
			return nil
		}
	}

	fmt.Println()
	fmt.Println(ui.TitleStyle.Render("🔗 Link to existing Vaulty vault"))
	fmt.Println()

	var repoInput string
	err = huh.NewInput().
		Title("Repository").
		Placeholder("owner/repo").
		Value(&repoInput).
		Validate(func(s string) error {
			if s == "" {
				return fmt.Errorf("repository is required")
			}
			_, _, err := github.ParseRepo(s)
			return err
		}).
		Run()
	if err != nil {
		return fmt.Errorf("form cancelled")
	}

	owner, repo, _ := github.ParseRepo(repoInput)
	repoFull := fmt.Sprintf("%s/%s", owner, repo)

	token, err := github.GetGitHubToken()
	if err != nil {
		return fmt.Errorf("GitHub authentication: %w", err)
	}

	client := github.NewClient(token)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println()
	fmt.Println(ui.MutedStyle.Render("Fetching vault metadata..."))

	metadataResp, err := client.GetContent(ctx, owner, repo, "metadata.vty")
	if err != nil {

		metadataResp, err = client.GetContent(ctx, owner, repo, "metadata.json")
		if err != nil {
			return fmt.Errorf("fetching vault metadata: %w", err)
		}
	}

	metadataEncData, err := client.DecodeContent(metadataResp)
	if err != nil {
		return fmt.Errorf("decoding metadata: %w", err)
	}

	var metadata config.Metadata
	if err := json.Unmarshal(metadataEncData, &metadata); err != nil {
		return fmt.Errorf("parsing metadata: %w", err)
	}

	cfg.SetRepo(repoFull)
	cfg.Metadata = &metadata

	if err := cfg.Save(""); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Println()
	fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("✅ Linked to vault: %s", repoFull)))
	fmt.Println()
	fmt.Println(ui.InfoStyle.Render("Available users:"))
	for _, user := range metadata.Users {
		fmt.Printf("  • %s (%s)\n", user.Username, user.Role)
	}
	fmt.Println()
	fmt.Println(ui.MutedStyle.Render("Run 'vty login' to authenticate and access the vault."))

	return nil
}

func init() {
	rootCmd.AddCommand(linkCmd)
}
