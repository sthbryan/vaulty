package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/spf13/cobra"
	"github.com/sthbryan/vaulty/internal/config"
	"github.com/sthbryan/vaulty/internal/github"
	"github.com/sthbryan/vaulty/internal/ui"
	"github.com/sthbryan/vaulty/pkg/models"
)

var (
	listType string
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "📊 List all secrets in the vault",
	Long: `List all secrets stored in your Vaulty repository.

This command retrieves and displays all environment files and SSH keys
stored in your configured GitHub repository with their metadata.

Examples:
  vty list              # List all secrets
  vty list --type=env   # List only environment secrets
  vty list --type=ssh   # List only SSH keys`,
	RunE: runList,
}

type ListItem struct {
	Name     string
	Type     string
	Modified time.Time
	Size     int64
}

func runList(cmd *cobra.Command, args []string) error {

	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	token, err := github.GetGitHubToken()
	if err != nil {
		return fmt.Errorf("failed to get GitHub token: %w", err)
	}

	owner, repo, err := github.ParseRepo(cfg.Repo)
	if err != nil {
		return fmt.Errorf("invalid repo format: %w", err)
	}

	client := github.NewClient(token)
	ctx := context.Background()

	fmt.Println()
	fmt.Println(ui.InfoStyle.Render("🔐 Fetching secrets from GitHub..."))
	fmt.Println()

	var items []ListItem

	if listType == "all" || listType == "env" {
		envItems, err := listDirectory(ctx, client, owner, repo, "envs", "env")
		if err != nil {
			ui.PrintWarning("Could not list envs directory: %v", err)
		} else {
			items = append(items, envItems...)
		}
	}

	if listType == "all" || listType == "ssh" {
		sshItems, err := listDirectory(ctx, client, owner, repo, "ssh", "ssh")
		if err != nil {
			ui.PrintWarning("Could not list ssh directory: %v", err)
		} else {
			items = append(items, sshItems...)
		}
	}

	if len(items) == 0 {
		fmt.Println(ui.MutedStyle.Render("No secrets found in vault"))
		fmt.Println()
		return nil
	}

	renderTable(items)

	fmt.Println()
	fmt.Println(ui.InfoStyle.Render(fmt.Sprintf("📊 Total: %d secret(s)", len(items))))
	fmt.Println()

	return nil
}

func listDirectory(ctx context.Context, client *github.Client, owner, repo, dirPath, itemType string) ([]ListItem, error) {
	var items []ListItem

	dirItems, err := client.ListDirectory(ctx, owner, repo, dirPath)
	if err != nil {

		if strings.Contains(err.Error(), "404") {
			return items, nil
		}
		return items, err
	}

	for _, item := range dirItems {

		if item.Type != "file" {
			continue
		}

		name := item.Name
		if strings.HasSuffix(name, ".json") {
			name = strings.TrimSuffix(name, ".json")
		}

		listItem := ListItem{
			Name: name,
			Type: itemType,
			Size: int64(item.Size),
		}

		content, err := client.GetContent(ctx, owner, repo, item.Path)
		if err == nil && content != nil {

			decoded, err := client.DecodeContent(content)
			if err == nil {

				var vaultFile models.VaultFile
				if err := json.Unmarshal(decoded, &vaultFile); err == nil {
					listItem.Modified = vaultFile.Metadata.UpdatedAt
					listItem.Size = vaultFile.Metadata.Size
				}
			}
		}

		items = append(items, listItem)
	}

	return items, nil
}

func renderTable(items []ListItem) {

	primary := lipgloss.Color("#7C3AED")
	success := lipgloss.Color("#10B981")
	muted := lipgloss.Color("#6B7280")

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(primary).
		Padding(0, 1)

	cellStyle := lipgloss.NewStyle().
		Padding(0, 1)

	envStyle := lipgloss.NewStyle().
		Foreground(success).
		Bold(true)

	sshStyle := lipgloss.NewStyle().
		Foreground(primary).
		Bold(true)

	rows := [][]string{
		{headerStyle.Render("Name"), headerStyle.Render("Type"), headerStyle.Render("Modified"), headerStyle.Render("Size")},
	}

	for _, item := range items {

		var typeStr string
		if item.Type == "env" {
			typeStr = envStyle.Render("🔐 env")
		} else {
			typeStr = sshStyle.Render("🔑 ssh")
		}

		modifiedStr := "Unknown"
		if !item.Modified.IsZero() {
			modifiedStr = item.Modified.Format("2006-01-02 15:04")
		}

		sizeStr := ui.FormatBytes(item.Size)

		rows = append(rows, []string{
			cellStyle.Render(item.Name),
			typeStr,
			cellStyle.Render(modifiedStr),
			cellStyle.Render(sizeStr),
		})
	}

	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(muted)).
		Rows(rows...)

	fmt.Println(t.Render())
}

func init() {
	listCmd.Flags().StringVar(&listType, "type", "all", "Filter by type: env, ssh, or all")
	rootCmd.AddCommand(listCmd)
}
