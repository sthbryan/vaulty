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
	if len(users) == 0 {
		return []string{}
	}

	maxUserWidth := 0
	maxRoleWidth := 0
	for _, u := range users {
		if len(u.Username) > maxUserWidth {
			maxUserWidth = len(u.Username)
		}
		if len(u.Role) > maxRoleWidth {
			maxRoleWidth = len(u.Role)
		}
	}

	var lines []string
	for _, u := range users {
		lines = append(lines, fmt.Sprintf("%s %s %s",
			padRight(ui.HighlightStyle.Render(u.Username), maxUserWidth),
			padRight(u.Role, maxRoleWidth),
			formatTime(u.CreatedAt)))
	}
	return lines
}

func buildVaultLines(secrets []models.SecretInfo) []string {
	if len(secrets) == 0 {
		return []string{}
	}

	nameWidth := 0
	typeWidth := 0
	envWidth := 0
	sizeWidth := 0

	for _, s := range secrets {
		if len(s.Name) > nameWidth {
			nameWidth = len(s.Name)
		}
		typeLen := len(string(s.Type))
		if typeLen > typeWidth {
			typeWidth = typeLen
		}
		envLen := 10
		if s.Type != models.SecretTypeEnv {
			envLen = 1
		}
		if envLen > envWidth {
			envWidth = envLen
		}
		sizeLen := len(formatSize(s.Size))
		if sizeLen > sizeWidth {
			sizeWidth = sizeLen
		}
	}

	var lines []string
	for _, s := range secrets {
		envDisplay := "-"
		line := fmt.Sprintf("%s %s %s %s %s",
			padRight(s.Name, nameWidth),
			padRight(ui.HighlightStyle.Render(string(s.Type)), typeWidth),
			padRight(envDisplay, envWidth),
			padRight(formatSize(s.Size), sizeWidth),
			formatTime(s.UpdatedAt))
		lines = append(lines, line)
	}
	return lines
}

func buildSSHKeyLines(keys []github.SSHKeyInfo, role string) []string {
	if len(keys) == 0 {
		return []string{}
	}

	var lines []string
	if role == "owner" {
		maxUserWidth := 0
		maxKeyWidth := 0
		for _, k := range keys {
			if len(k.Username) > maxUserWidth {
				maxUserWidth = len(k.Username)
			}
			if len(k.KeyName) > maxKeyWidth {
				maxKeyWidth = len(k.KeyName)
			}
		}
		for _, k := range keys {
			lines = append(lines, fmt.Sprintf("%s %s %s",
				padRight(ui.HighlightStyle.Render(k.Username), maxUserWidth),
				padRight(k.KeyName, maxKeyWidth),
				formatSize(int64(k.Size))))
		}
	} else {
		maxKeyWidth := 0
		for _, k := range keys {
			if len(k.KeyName) > maxKeyWidth {
				maxKeyWidth = len(k.KeyName)
			}
		}
		for _, k := range keys {
			lines = append(lines, fmt.Sprintf("%s %s",
				padRight(k.KeyName, maxKeyWidth),
				formatSize(int64(k.Size))))
		}
	}
	return lines
}

func buildVaultInfoLines(cfg *config.Config, sess *session.Session, sshKeys []github.SSHKeyInfo, lastSync time.Time, vaultPath string) []string {
	// Calculate max key width for alignment
	maxKeyWidth := 0
	var allLines []string

	if sess.Role == "owner" {
		allLines = append(allLines, "[@] SSH Breakdown:")
		userKeyCounts := make(map[string]int)
		userKeySizes := make(map[string]int64)
		for _, key := range sshKeys {
			userKeyCounts[key.Username]++
			userKeySizes[key.Username] += int64(key.Size)
		}
		for username, count := range userKeyCounts {
			line := fmt.Sprintf("  %s: %s (%s)",
				ui.HighlightStyle.Render(username),
				ui.HighlightStyle.Render(fmt.Sprintf("%d keys", count)),
				ui.HighlightStyle.Render(formatSize(userKeySizes[username])))
			allLines = append(allLines, line)
			if len("  "+username) > maxKeyWidth {
				maxKeyWidth = len("  " + username)
			}
		}
	} else {
		allLines = append(allLines, fmt.Sprintf("[@] SSH Keys: %s",
			ui.HighlightStyle.Render(fmt.Sprintf("%d", len(sshKeys)))))
		if len("[@] SSH Keys") > maxKeyWidth {
			maxKeyWidth = len("[@] SSH Keys")
		}
	}

	if sess.Role == "owner" && cfg.Metadata != nil {
		ownerCount, editorCount, viewerCount := countUsersByRole(cfg.Metadata.Users)
		line := fmt.Sprintf("[U] Users: %s total (%d own, %d ed, %d view)",
			ui.HighlightStyle.Render(fmt.Sprintf("%d", len(cfg.Metadata.Users))),
			ownerCount, editorCount, viewerCount)
		allLines = append(allLines, line)
		if len("[U] Users") > maxKeyWidth {
			maxKeyWidth = len("[U] Users")
		}
	}

	if cfg.IsLocalMode() {
		allLines = append(allLines, fmt.Sprintf("[L] Vault:   %s", ui.HighlightStyle.Render("local")))
		allLines = append(allLines, fmt.Sprintf("[R] Path:    %s", vaultPath))
	} else {
		allLines = append(allLines, fmt.Sprintf("[G] Vault:   %s", ui.HighlightStyle.Render("github")))
		allLines = append(allLines, fmt.Sprintf("[R] Repo:    %s", cfg.Repo))
	}

	allLines = append(allLines, fmt.Sprintf("[<] Sync:    %s", ui.HighlightStyle.Render(formatTime(lastSync))))
	allLines = append(allLines, fmt.Sprintf("[^] Updt:    %s", ui.HighlightStyle.Render(formatTime(cfg.UpdatedAt))))
	allLines = append(allLines, fmt.Sprintf("[+] Created: %s", ui.HighlightStyle.Render(formatTime(cfg.CreatedAt))))

	// Update maxKeyWidth with remaining keys
	for _, key := range []string{"[L] Vault", "[R] Path", "[<] Sync", "[^] Updt", "[+] Created"} {
		if len(key) > maxKeyWidth {
			maxKeyWidth = len(key)
		}
	}

	// Rebuild with alignment
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
			lines = append(lines, fmt.Sprintf("  %s %s (%s)",
				padRight(ui.HighlightStyle.Render(username), maxKeyWidth),
				padRight(ui.HighlightStyle.Render(fmt.Sprintf("%d keys", count)), 8),
				ui.HighlightStyle.Render(formatSize(userKeySizes[username]))))
		}
	} else {
		lines = append(allLines, fmt.Sprintf("[@] SSH Keys: %s",
			ui.HighlightStyle.Render(fmt.Sprintf("%d", len(sshKeys)))))
	}

	if sess.Role == "owner" && cfg.Metadata != nil {
		ownerCount, editorCount, viewerCount := countUsersByRole(cfg.Metadata.Users)
		lines = append(lines, fmt.Sprintf("%s %s total (%d own, %d ed, %d view)",
			padRight("[U] Users", maxKeyWidth),
			ui.HighlightStyle.Render(fmt.Sprintf("%d", len(cfg.Metadata.Users))),
			ownerCount, editorCount, viewerCount))
	}

	if cfg.IsLocalMode() {
		lines = append(lines, fmt.Sprintf("%s %s", padRight("[L] Vault", maxKeyWidth), ui.HighlightStyle.Render("local")))
		lines = append(lines, fmt.Sprintf("%s %s", padRight("[R] Path", maxKeyWidth), vaultPath))
	} else {
		lines = append(lines, fmt.Sprintf("%s %s", padRight("[G] Vault", maxKeyWidth), ui.HighlightStyle.Render("github")))
		lines = append(lines, fmt.Sprintf("%s %s", padRight("[R] Repo", maxKeyWidth), cfg.Repo))
	}

	lines = append(lines, fmt.Sprintf("%s %s", padRight("[<] Sync", maxKeyWidth), ui.HighlightStyle.Render(formatTime(lastSync))))
	lines = append(lines, fmt.Sprintf("%s %s", padRight("[^] Updt", maxKeyWidth), ui.HighlightStyle.Render(formatTime(cfg.UpdatedAt))))
	lines = append(lines, fmt.Sprintf("%s %s", padRight("[+] Created", maxKeyWidth), ui.HighlightStyle.Render(formatTime(cfg.CreatedAt))))

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

func padRight(s string, width int) string {
	stripped := stripANSI(s)
	padding := width - len(stripped)
	if padding < 0 {
		padding = 0
	}
	return s + strings.Repeat(" ", padding)
}
