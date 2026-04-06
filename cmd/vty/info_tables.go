package main

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/sthbryan/vaulty/internal/config"
	"github.com/sthbryan/vaulty/internal/github"
	"github.com/sthbryan/vaulty/internal/ui"
	"github.com/sthbryan/vaulty/pkg/models"
)

func renderUsersTable(users []config.UserEntry) {
	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("240"))).
		Headers("USERNAME", "ROLE", "CREATED")

	for _, user := range users {
		t.Row(
			user.Username,
			user.Role,
			formatTime(user.CreatedAt),
		)
	}

	fmt.Println(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(ui.Primary)).Render("=== USERS ==="))
	fmt.Println(t.Render())
}

func renderSecretsTable(secrets []models.SecretInfo) {
	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("240"))).
		Headers("NAME", "TYPE", "ENVIRONMENT", "SIZE", "UPDATED")

	for _, secret := range secrets {
		env := secret.Environment
		typeColor := lipgloss.Color("white")

		switch secret.Type {
		case models.SecretTypeEnv:
			typeColor = lipgloss.Color("76")
			if env == "" {
				env = "shared"
			}
			env = envBadge(env)
		case models.SecretTypeResource:
			typeColor = lipgloss.Color("75")
			env = "-"
		case models.SecretTypeConfig:
			typeColor = lipgloss.Color("220")
			env = "-"
		default:
			env = "-"
		}

		t.Row(
			secret.Name,
			lipgloss.NewStyle().Foreground(typeColor).Render(string(secret.Type)),
			env,
			formatSize(secret.Size),
			formatTime(secret.UpdatedAt),
		)
	}

	fmt.Println(t.Render())
}

func envBadge(env string) string {
	switch env {
	case "shared":
		return "[*] shared"
	default:
		return "[@] " + env
	}
}

func renderSSHKeysTable(keys []github.SSHKeyInfo, role string) {
	if role == "owner" {
		t := table.New().
			Border(lipgloss.NormalBorder()).
			BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("240"))).
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
