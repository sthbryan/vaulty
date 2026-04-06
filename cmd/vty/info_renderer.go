package main

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/sthbryan/vaulty/internal/config"
	"github.com/sthbryan/vaulty/internal/github"
	"github.com/sthbryan/vaulty/internal/session"
	"github.com/sthbryan/vaulty/internal/ui"
	"github.com/sthbryan/vaulty/pkg/models"
)

func renderDetailedVaultInfo(
	cfg *config.Config,
	sess *session.Session,
	secrets []models.SecretInfo,
	sshKeys []github.SSHKeyInfo,
	lastSync time.Time,
	vaultPath string,
) {
	fmt.Println()
	fmt.Println(ui.MutedStyle.Render("User: " + sess.Username + " (" + sess.Role + ")"))
	fmt.Println()

	if sess.Role == "owner" && cfg.Metadata != nil && len(cfg.Metadata.Users) > 0 {
		lines := buildUserLines(cfg.Metadata.Users)
		renderPanel("USERS", lines)
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
			if vaultSecrets[i].Type != vaultSecrets[j].Type {
				return vaultSecrets[i].Type < vaultSecrets[j].Type
			}
			if vaultSecrets[i].Environment != vaultSecrets[j].Environment {
				return vaultSecrets[i].Environment < vaultSecrets[j].Environment
			}
			return vaultSecrets[i].Name < vaultSecrets[j].Name
		})
		lines := buildVaultLines(secrets)
		renderPanel("VAULT", lines)
		fmt.Println()
	}

	if len(sshKeys) > 0 {
		lines := buildSSHKeyLines(sshKeys, sess.Role)
		renderPanel("SSH KEYS", lines)
		fmt.Println()
	}

	secretsLines := []string{
		fmt.Sprintf("[*] Envs:     %s", ui.HighlightStyle.Render(fmt.Sprintf("%d", countSecretsByType(secrets, models.SecretTypeEnv)))),
		fmt.Sprintf("[>] Res:      %s", ui.HighlightStyle.Render(fmt.Sprintf("%d", countSecretsByType(secrets, models.SecretTypeResource)))),
		fmt.Sprintf("[~] Cfg:      %s", ui.HighlightStyle.Render(fmt.Sprintf("%d", countSecretsByType(secrets, models.SecretTypeConfig)))),
		fmt.Sprintf("[@] SSH Keys: %s", ui.HighlightStyle.Render(fmt.Sprintf("%d", len(sshKeys)))),
		fmt.Sprintf("[=] Size:     %s", ui.HighlightStyle.Render(formatSize(sumSecretSizes(secrets)))),
	}
	renderPanel("SECRETS", secretsLines)
	fmt.Println()

	vaultInfoLines := buildVaultInfoLines(cfg, sess, sshKeys, lastSync, vaultPath)
	renderPanel("INFO", vaultInfoLines)
	fmt.Println()
}

func buildUserLines(users []config.UserEntry) []string {
	var lines []string
	for _, u := range users {
		lines = append(lines, fmt.Sprintf("%s  %s  %s",
			ui.HighlightStyle.Render(u.Username),
			u.Role,
			formatTime(u.CreatedAt)))
	}
	return lines
}

func buildVaultLines(secrets []models.SecretInfo) []string {
	var lines []string
	for _, s := range secrets {
		envDisplay := "-"
		if s.Type == models.SecretTypeEnv && s.Environment != "" {
			envDisplay = envBadge(s.Environment)
		}
		lines = append(lines, fmt.Sprintf("%s  %s  %s  %s  %s",
			s.Name,
			ui.HighlightStyle.Render(string(s.Type)),
			envDisplay,
			formatSize(s.Size),
			formatTime(s.UpdatedAt)))
	}
	return lines
}

func buildSSHKeyLines(keys []github.SSHKeyInfo, role string) []string {
	var lines []string
	if role == "owner" {
		for _, k := range keys {
			lines = append(lines, fmt.Sprintf("%s  %s  %s",
				ui.HighlightStyle.Render(k.Username),
				k.KeyName,
				formatSize(int64(k.Size))))
		}
	} else {
		for _, k := range keys {
			lines = append(lines, fmt.Sprintf("%s  %s",
				k.KeyName,
				formatSize(int64(k.Size))))
		}
	}
	return lines
}

func buildVaultInfoLines(cfg *config.Config, sess *session.Session, sshKeys []github.SSHKeyInfo, lastSync time.Time, vaultPath string) []string {
	var lines []string

	if sess.Role == "owner" {
		lines = append(lines, "[@] SSH Breakdown:")
		userKeyCounts := make(map[string]int)
		userKeySizes := make(map[string]int64)
		for _, key := range sshKeys {
			userKeyCounts[key.Username]++
			userKeySizes[key.Username] += int64(key.Size)
		}
		for username, count := range userKeyCounts {
			lines = append(lines, fmt.Sprintf("  %s: %s (%s)",
				ui.HighlightStyle.Render(username),
				ui.HighlightStyle.Render(fmt.Sprintf("%d keys", count)),
				ui.HighlightStyle.Render(formatSize(userKeySizes[username]))))
		}
	} else {
		lines = append(lines, fmt.Sprintf("[@] SSH Keys: %s",
			ui.HighlightStyle.Render(fmt.Sprintf("%d", len(sshKeys)))))
	}

	if sess.Role == "owner" && cfg.Metadata != nil {
		ownerCount, editorCount, viewerCount := countUsersByRole(cfg.Metadata.Users)
		lines = append(lines, fmt.Sprintf("[U] Users: %s total (%d own, %d ed, %d view)",
			ui.HighlightStyle.Render(fmt.Sprintf("%d", len(cfg.Metadata.Users))),
			ownerCount, editorCount, viewerCount))
	}

	if cfg.IsLocalMode() {
		lines = append(lines, fmt.Sprintf("[L] Vault:   %s", ui.HighlightStyle.Render("local")))
		lines = append(lines, fmt.Sprintf("[R] Path:    %s", vaultPath))
	} else {
		lines = append(lines, fmt.Sprintf("[G] Vault:   %s", ui.HighlightStyle.Render("github")))
		lines = append(lines, fmt.Sprintf("[R] Repo:    %s", cfg.Repo))
	}

	lines = append(lines, fmt.Sprintf("[<] Sync:    %s", ui.HighlightStyle.Render(formatTime(lastSync))))
	lines = append(lines, fmt.Sprintf("[^] Updt:    %s", ui.HighlightStyle.Render(formatTime(cfg.UpdatedAt))))
	lines = append(lines, fmt.Sprintf("[+] Created: %s", ui.HighlightStyle.Render(formatTime(cfg.CreatedAt))))

	return lines
}

func countSecretsByType(secrets []models.SecretInfo, secretType models.SecretType) int {
	count := 0
	for _, s := range secrets {
		if s.Type == secretType {
			count++
		}
	}
	return count
}

func sumSecretSizes(secrets []models.SecretInfo) int64 {
	var total int64
	for _, s := range secrets {
		total += s.Size
	}
	return total
}

func countUsersByRole(users []config.UserEntry) (owners, editors, viewers int) {
	for _, u := range users {
		switch u.Role {
		case "owner":
			owners++
		case "editor":
			editors++
		case "viewer":
			viewers++
		}
	}
	return
}

func envBadge(env string) string {
	switch env {
	case "shared":
		return "[*] shared"
	default:
		return "[@] " + env
	}
}

func formatSize(bytes int64) string {
	const B = 1
	const KB = 1024 * B
	const MB = 1024 * KB
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
		return "not synced"
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

func stripANSI(s string) string {
	var result strings.Builder
	inANSI := false
	for _, c := range s {
		if c == '\x1b' {
			inANSI = true
		} else if c == 'm' && inANSI {
			inANSI = false
		} else if !inANSI {
			result.WriteRune(c)
		}
	}
	return result.String()
}
