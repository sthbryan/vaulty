package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/DeadBryam/vaulty/internal/config"
	"github.com/DeadBryam/vaulty/internal/crypto"
	"github.com/DeadBryam/vaulty/internal/github"
	"github.com/DeadBryam/vaulty/internal/session"
	"github.com/DeadBryam/vaulty/internal/ui"
	"github.com/DeadBryam/vaulty/pkg/models"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show vault contents and metadata",
	Long: `Display all secrets stored in your Vaulty vault.

Shows name, type, size, and when each secret was last updated.
Requires an active session (use 'vty unlock' first).`,
	RunE: runInfo,
}

func runInfo(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if cfg.Repo == "" {
		return fmt.Errorf("Vaulty not initialized. Run 'vty init' first")
	}

	mgr := session.GetManager()
	currentSession := mgr.Get(cfg.CurrentUser)
	if currentSession == nil || currentSession.MasterKey == nil {
		return fmt.Errorf("no active session. Run 'vty unlock' first")
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
	fmt.Println(ui.MutedStyle.Render("Fetching vault contents..."))

	vaultResp, err := client.GetContent(ctx, owner, repo, ".vaulty/vault.enc")
	if err != nil {
		return fmt.Errorf("failed to fetch vault: %w", err)
	}

	vaultEncData, err := client.DecodeContent(vaultResp)
	if err != nil {
		return fmt.Errorf("decoding vault: %w", err)
	}

	masterKey := currentSession.MasterKey
	vaultData, err := crypto.DecryptVaultData(&crypto.EncryptedData{
		Salt:       []byte(vaultEncData[:32]),
		IV:         []byte(vaultEncData[32:48]),
		Ciphertext: []byte(vaultEncData[48:]),
	}, masterKey)
	if err != nil {
		return fmt.Errorf("decrypting vault: %w", err)
	}

	var vaultContents map[string]models.VaultFile
	if err := json.Unmarshal(vaultData, &vaultContents); err != nil {
		return fmt.Errorf("parsing vault: %w", err)
	}

	if len(vaultContents) == 0 {
		fmt.Println()
		fmt.Println(ui.InfoStyle.Render("No secrets found in vault"))
		return nil
	}

	var secrets []models.SecretInfo
	for name, vaultFile := range vaultContents {
		secrets = append(secrets, models.SecretInfo{
			Name:      name,
			Type:      vaultFile.Metadata.Type,
			CreatedAt: vaultFile.Metadata.CreatedAt,
			UpdatedAt: vaultFile.Metadata.UpdatedAt,
			Size:      vaultFile.Metadata.Size,
		})
	}

	// Sort secrets by type then name
	sort.Slice(secrets, func(i, j int) bool {
		if secrets[i].Type == secrets[j].Type {
			return secrets[i].Name < secrets[j].Name
		}
		return secrets[i].Type < secrets[j].Type
	})

	renderDetailedVaultInfo(cfg, secrets, cfg.UpdatedAt)
	return nil
}

func renderDetailedVaultInfo(cfg *config.Config, secrets []models.SecretInfo, lastSync time.Time) {
	fmt.Println()

	// User info
	fmt.Println(ui.MutedStyle.Render("User: " + cfg.CurrentUser + " (" + cfg.CurrentUserRole + ")"))
	fmt.Println()

	// Separate secrets by type
	var envSecrets, sshSecrets []models.SecretInfo
	for _, s := range secrets {
		if s.Type == models.SecretTypeSSH {
			sshSecrets = append(sshSecrets, s)
		} else {
			envSecrets = append(envSecrets, s)
		}
	}

	// Render env vars table
	if len(envSecrets) > 0 {
		fmt.Println(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(ui.Primary)).Render("=== ENVIRONMENT VARIABLES ==="))
		renderSecretsTable(envSecrets)
		fmt.Println()
	}

	// Render ssh keys table
	if len(sshSecrets) > 0 {
		fmt.Println(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(ui.Primary)).Render("=== SSH KEYS ==="))
		renderSecretsTable(sshSecrets)
		fmt.Println()
	}

	// Render summary
	fmt.Println(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(ui.Primary)).Render("=== SUMMARY ==="))
	totalSize := int64(0)
	for _, s := range secrets {
		totalSize += s.Size
	}

	fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Total Secrets: %d", len(secrets))))
	fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Total Size: %s", formatSize(totalSize))))
	fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Last Sync: %s", formatTime(lastSync))))
	fmt.Println()
}

func renderSecretsTable(secrets []models.SecretInfo) {
	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("240"))).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == 0 {
				return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(ui.Primary))
			}
			if row%2 == 0 {
				return lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
			}
			return lipgloss.NewStyle()
		}).
		Headers("NAME", "TYPE", "SIZE", "UPDATED")

	for _, secret := range secrets {
		t.Row(
			secret.Name,
			string(secret.Type),
			formatSize(secret.Size),
			formatTime(secret.UpdatedAt),
		)
	}

	fmt.Println(t.Render())
}

func formatSize(bytes int64) string {
	const (
		B  = 1
		KB = 1024 * B
		MB = 1024 * KB
	)
	switch {
	case bytes < KB:
		return fmt.Sprintf("%dB", bytes)
	case bytes < MB:
		return fmt.Sprintf("%.1fKB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%.1fMB", float64(bytes)/float64(MB))
	}
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "never"
	}
	duration := time.Since(t)
	if duration < time.Minute {
		return "just now"
	}
	if duration < time.Hour {
		return fmt.Sprintf("%d mins ago", int(duration.Minutes()))
	}
	if duration < 24*time.Hour {
		return fmt.Sprintf("%d hours ago", int(duration.Hours()))
	}
	days := int(duration.Hours()) / 24
	if days == 1 {
		return "1 day ago"
	}
	if days < 7 {
		return fmt.Sprintf("%d days ago", days)
	}
	weeks := days / 7
	if weeks == 1 {
		return "1 week ago"
	}
	if weeks < 4 {
		return fmt.Sprintf("%d weeks ago", weeks)
	}
	return t.Format("2006-01-02")
}

func init() {
	rootCmd.AddCommand(infoCmd)
}
