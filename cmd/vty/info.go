package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
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
Requires an active session (use 'vty login' first).`,
	RunE: runInfo,
}

func fetchAndDecryptVtyFile(ctx context.Context, client *github.Client, owner, repo, path string, masterKey []byte) ([]byte, error) {
	content, err := client.GetContent(ctx, owner, repo, path)
	if err != nil {
		return nil, fmt.Errorf("fetching content: %w", err)
	}

	encodedData, err := client.DecodeContent(content)
	if err != nil {
		return nil, fmt.Errorf("decoding content: %w", err)
	}

	hexData := string(encodedData)
	plaintext, err := crypto.DecryptBinary(hexData, masterKey)
	if err != nil {
		if err == crypto.ErrDecryptionFailed {
			return nil, fmt.Errorf("decryption failed: invalid password")
		}
		return nil, fmt.Errorf("decrypting: %w", err)
	}

	return plaintext, nil
}

func runInfo(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	sess, err := ensureAuthenticated(cfg)
	if err != nil {
		return err
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

	var secrets []models.SecretInfo
	var sshKeys []github.SSHKeyInfo

	// Fetch and decrypt vault.vty to get metadata
	logger.Info("🔓 Decrypting vault...")
	vaultContent, err := fetchAndDecryptVtyFile(ctx, client, owner, repo, ".vaulty/vault.vty", sess.MasterKey)
	if err != nil {
		logger.Warn("Could not decrypt vault", "error", err)
	}

	var vaultData map[string]interface{}
	if vaultContent != nil {
		if err := json.Unmarshal(vaultContent, &vaultData); err != nil {
			logger.Warn("Could not parse vault data", "error", err)
		}
	}

	// Fetch and decrypt environment secrets
	envItems, err := client.ListDirectory(ctx, owner, repo, "envs")
	if err == nil {
		for _, item := range envItems {
			if strings.HasSuffix(item.Name, ".vty") {
				name := strings.TrimSuffix(item.Name, ".vty")
				path := fmt.Sprintf("envs/%s", item.Name)

				// Fetch and decrypt to get actual size
				decryptedContent, decryptErr := fetchAndDecryptVtyFile(ctx, client, owner, repo, path, sess.MasterKey)
				var size int64
				if decryptErr == nil {
					size = int64(len(decryptedContent))
				} else {
					size = int64(item.Size)
					logger.Debug("Could not decrypt env file", "name", name, "error", decryptErr)
				}

				secrets = append(secrets, models.SecretInfo{
					Name:      name,
					Type:      models.SecretTypeEnv,
					CreatedAt: time.Time{},
					UpdatedAt: time.Time{},
					Size:      size,
				})
			}
		}
	}

	if sess.Role == "owner" {
		sshKeys, err = client.ListAllSSHKeys(ctx, owner, repo)
	} else {
		sshKeys, err = client.ListSSHKeys(ctx, owner, repo, sess.Username)
	}
	if err == nil {
		for _, key := range sshKeys {
			// Fetch and decrypt SSH key to get actual size
			path := fmt.Sprintf("ssh/%s/%s.vty", key.Username, key.KeyName)
			decryptedContent, decryptErr := fetchAndDecryptVtyFile(ctx, client, owner, repo, path, sess.MasterKey)

			var size int64
			if decryptErr == nil {
				size = int64(len(decryptedContent))
			} else {
				size = int64(key.Size)
				logger.Debug("Could not decrypt SSH key", "name", key.KeyName, "error", decryptErr)
			}

			secrets = append(secrets, models.SecretInfo{
				Name:      key.KeyName,
				Type:      models.SecretTypeSSH,
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
				Size:      size,
			})
		}
	}

	if len(secrets) == 0 {
		fmt.Println()
		fmt.Println(ui.InfoStyle.Render("No secrets found in vault"))
		return nil
	}

	sort.Slice(secrets, func(i, j int) bool {
		if secrets[i].Type == secrets[j].Type {
			return secrets[i].Name < secrets[j].Name
		}
		return secrets[i].Type < secrets[j].Type
	})

	renderDetailedVaultInfo(cfg, sess, secrets, sshKeys, cfg.UpdatedAt)
	return nil
}

func renderDetailedVaultInfo(cfg *config.Config, sess *session.Session, secrets []models.SecretInfo, sshKeys []github.SSHKeyInfo, lastSync time.Time) {
	fmt.Println()

	fmt.Println(ui.MutedStyle.Render("User: " + sess.Username + " (" + sess.Role + ")"))
	fmt.Println()

	if sess.Role == "owner" && cfg.Metadata != nil && len(cfg.Metadata.Users) > 0 {
		fmt.Println(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(ui.Primary)).Render("=== USERS ==="))
		renderUsersTable(cfg.Metadata.Users)
		fmt.Println()
	}

	var envSecrets, sshSecrets []models.SecretInfo
	for _, s := range secrets {
		if s.Type == models.SecretTypeSSH {
			sshSecrets = append(sshSecrets, s)
		} else {
			envSecrets = append(envSecrets, s)
		}
	}

	if len(envSecrets) > 0 {
		fmt.Println(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(ui.Primary)).Render("=== ENVIRONMENT VARIABLES ==="))
		renderSecretsTable(envSecrets)
		fmt.Println()
	}

	if len(sshKeys) > 0 {
		fmt.Println(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(ui.Primary)).Render("=== SSH KEYS ==="))
		renderSSHKeysTable(sshKeys, sess.Role)
		fmt.Println()
	}

	totalSize := int64(0)
	envCount := 0
	sshCount := len(sshKeys)
	for _, s := range secrets {
		totalSize += s.Size
		if s.Type == models.SecretTypeEnv {
			envCount++
		}
	}

	fmt.Println(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(ui.Primary)).Render("=== SUMMARY ==="))
	fmt.Println()

	fmt.Printf("  Total Secrets: %s (", ui.HighlightStyle.Render(fmt.Sprintf("%d", len(secrets))))
	fmt.Printf("%s env + ", ui.HighlightStyle.Render(fmt.Sprintf("%d", envCount)))
	fmt.Printf("%s ssh)\n", ui.HighlightStyle.Render(fmt.Sprintf("%d", sshCount)))
	fmt.Printf("  Total Size:    %s (", ui.HighlightStyle.Render(formatSize(totalSize)))
	fmt.Printf("%s env + ", ui.HighlightStyle.Render(formatSize(calculateTypeSize(secrets, models.SecretTypeEnv))))
	fmt.Printf("%s ssh)\n", ui.HighlightStyle.Render(formatSize(calculateTypeSize(secrets, models.SecretTypeSSH))))
	fmt.Println()

	if sess.Role == "owner" {
		fmt.Println("  SSH Breakdown:")
		userKeyCounts := make(map[string]int)
		userKeySizes := make(map[string]int64)
		for _, key := range sshKeys {
			userKeyCounts[key.Username]++
			userKeySizes[key.Username] += int64(key.Size)
		}
		for username, count := range userKeyCounts {
			fmt.Printf("    %s: %s (%s)\n",
				ui.HighlightStyle.Render(username),
				ui.HighlightStyle.Render(fmt.Sprintf("%d keys", count)),
				ui.HighlightStyle.Render(formatSize(userKeySizes[username])))
		}
		fmt.Println()
	} else {
		fmt.Printf("  My SSH Keys:   %s\n", ui.HighlightStyle.Render(fmt.Sprintf("%d", sshCount)))
		fmt.Println()
	}

	if sess.Role == "owner" && cfg.Metadata != nil {
		ownerCount := 0
		editorCount := 0
		viewerCount := 0
		for _, u := range cfg.Metadata.Users {
			switch u.Role {
			case "owner":
				ownerCount++
			case "editor":
				editorCount++
			case "viewer":
				viewerCount++
			}
		}
		fmt.Printf("  Users: %s total", ui.HighlightStyle.Render(fmt.Sprintf("%d", len(cfg.Metadata.Users))))
		if len(cfg.Metadata.Users) > 0 {
			fmt.Printf(" (%d owner, %d editor, %d viewer)", ownerCount, editorCount, viewerCount)
		}
		fmt.Println()
		fmt.Println()
	}

	fmt.Printf("  Repository:    %s\n", cfg.Repo)
	fmt.Printf("  Last Sync:     %s\n", ui.HighlightStyle.Render(formatTime(lastSync)))
	fmt.Printf("  Last Updated:  %s\n", ui.HighlightStyle.Render(formatTime(cfg.UpdatedAt)))
	fmt.Printf("  Created:       %s\n", ui.HighlightStyle.Render(formatTime(cfg.CreatedAt)))
	fmt.Println()
}

func renderUsersTable(users []config.UserEntry) {
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
		Headers("USERNAME", "ROLE", "CREATED")

	for _, user := range users {
		t.Row(
			user.Username,
			user.Role,
			formatTime(user.CreatedAt),
		)
	}

	fmt.Println(t.Render())
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

func renderSSHKeysTable(keys []github.SSHKeyInfo, role string) {
	if role == "owner" {
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
			Headers("USERNAME", "KEYNAME", "SIZE")

		for _, key := range keys {
			t.Row(
				key.Username,
				key.KeyName,
				formatSize(int64(key.Size)),
			)
		}

		fmt.Println(t.Render())
	} else {
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
			Headers("KEYNAME", "SIZE")

		for _, key := range keys {
			t.Row(
				key.KeyName,
				formatSize(int64(key.Size)),
			)
		}

		fmt.Println(t.Render())
	}
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

func calculateTypeSize(secrets []models.SecretInfo, secretType models.SecretType) int64 {
	var total int64
	for _, s := range secrets {
		if s.Type == secretType {
			total += s.Size
		}
	}
	return total
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
