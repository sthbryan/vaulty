package main

import (
	"fmt"
	"sort"
	"time"

	"github.com/DeadBryam/vaulty/internal/config"
	"github.com/DeadBryam/vaulty/internal/github"
	"github.com/DeadBryam/vaulty/internal/session"
	"github.com/DeadBryam/vaulty/internal/ui"
	"github.com/DeadBryam/vaulty/pkg/models"
	"github.com/charmbracelet/lipgloss"
)

func renderDetailedVaultInfo(cfg *config.Config, sess *session.Session, secrets []models.SecretInfo, sshKeys []github.SSHKeyInfo, lastSync time.Time) {
	fmt.Println()

	fmt.Println(ui.MutedStyle.Render("User: " + sess.Username + " (" + sess.Role + ")"))
	fmt.Println()

	if sess.Role == "owner" && cfg.Metadata != nil && len(cfg.Metadata.Users) > 0 {
		renderUsersTable(cfg.Metadata.Users)
		fmt.Println()
	}

	secretsByType := make(map[models.SecretType][]models.SecretInfo)
	for _, s := range secrets {
		secretsByType[s.Type] = append(secretsByType[s.Type], s)
	}

	var vaultSecrets []models.SecretInfo
	for _, t := range []models.SecretType{models.SecretTypeEnv, models.SecretTypeResource, models.SecretTypeConfig} {
		vaultSecrets = append(vaultSecrets, secretsByType[t]...)
	}
	if len(vaultSecrets) > 0 {
		sort.Slice(vaultSecrets, func(i, j int) bool {
			if vaultSecrets[i].Environment == vaultSecrets[j].Environment {
				return vaultSecrets[i].Name < vaultSecrets[j].Name
			}
			return vaultSecrets[i].Environment < vaultSecrets[j].Environment
		})
		fmt.Println(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(ui.Primary)).Render("=== VAULT ==="))
		renderSecretsTable(vaultSecrets)
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
