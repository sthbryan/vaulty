package main

import (
	"context"
	"encoding/json"
	"fmt"
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

	renderSecretTable(secrets)
	return nil
}

func renderSecretTable(secrets []models.SecretInfo) {
	fmt.Println()

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
			fmt.Sprintf("%d bytes", secret.Size),
			secret.UpdatedAt.Format("2006-01-02 15:04"),
		)
	}

	fmt.Println(t.Render())
	fmt.Println()
	fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Total: %d secrets", len(secrets))))
	fmt.Println()
}

func init() {
	rootCmd.AddCommand(infoCmd)
}
