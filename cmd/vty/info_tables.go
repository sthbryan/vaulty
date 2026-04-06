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
		Headers("NAME", "TYPE", "ENV", "TAG", "SIZE", "UPDATED")

	for _, secret := range secrets {
		env := secret.Environment
		tag := "-"

		switch secret.Type {
		case models.SecretTypeEnv:
			if env == "" {
				env = "shared"
			}
		case models.SecretTypeResource, models.SecretTypeConfig:
			if env != "" && env != "config" && env != "resources" {
				tag = env
				env = "-"
			} else {
				env = "-"
				tag = "-"
			}
		default:
			env = "-"
			tag = "-"
		}

		t.Row(
			secret.Name,
			string(secret.Type),
			env,
			tag,
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
