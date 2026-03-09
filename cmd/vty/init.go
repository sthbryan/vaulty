package main

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/sthbryan/vaulty/internal/config"
	"github.com/sthbryan/vaulty/internal/github"
	"github.com/sthbryan/vaulty/internal/ui"
)

const logo = `
██╗   ██╗ █████╗ ██╗   ██╗██╗  ████████╗██╗   ██╗
██║   ██║██╔══██╗██║   ██║██║  ╚══██╔══╝╚██╗ ██╔╝
██║   ██║███████║██║   ██║██║     ██║    ╚████╔╝ 
╚██╗ ██╔╝██╔══██║██║   ██║██║     ██║     ╚██╔╝  
 ╚████╔╝ ██║  ██║╚██████╔╝███████╗██║      ██║   
  ╚═══╝  ╚═╝  ╚═╝ ╚═════╝ ╚══════╝╚═╝      ╚═╝   
`

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Vaulty with a GitHub repository",
	Long: `Initialize Vaulty by creating or linking a GitHub repository.

This command will guide you through:
  • Creating a new private repository
  • Linking an existing repository
  • Configuring your local Vaulty settings`,
	RunE: runInit,
}

func runInit(cmd *cobra.Command, args []string) error {

	fmt.Println()
	fmt.Println(lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.Primary)).
		Bold(true).
		Render(logo))
	fmt.Println(ui.TitleStyle.Render("✨ Welcome to Vaulty!"))
	fmt.Println(ui.MutedStyle.Render("  Secure secret management powered by GitHub"))
	fmt.Println()

	var choice string
	err := huh.NewSelect[string]().
		Title("What would you like to do?").
		Options(
			huh.NewOption("Create a new repository", "create"),
			huh.NewOption("Link an existing repository", "link"),
		).
		Value(&choice).
		Run()
	if err != nil {
		return fmt.Errorf("form cancelled")
	}

	var repo string
	if choice == "create" {
		repo = createNewRepo()
	} else {
		repo = linkExistingRepo()
	}

	if repo == "" {
		return fmt.Errorf("no repository configured")
	}

	cfg := &config.Config{}
	cfg.SetRepo(repo)

	if err := cfg.Save(""); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Println()
	fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("✅ Vaulty initialized with repository: %s", repo)))
	fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("   Config saved to: %s", config.DefaultPath())))
	fmt.Println()

	return nil
}

func createNewRepo() string {
	fmt.Println()
	fmt.Println(ui.InfoStyle.Render("Creating a new private repository..."))
	fmt.Println()

	var name string
	for {
		err := huh.NewInput().
			Title("Repository name").
			Placeholder("my-vault-secrets").
			Value(&name).
			Validate(func(s string) error {
				if s == "" {
					return fmt.Errorf("name is required")
				}
				if !isValidRepoName(s) {
					return fmt.Errorf("invalid repository name")
				}
				return nil
			}).
			Run()
		if err != nil {
			return ""
		}
		break
	}

	username, err := getGitHubUsername()
	if err != nil {
		fmt.Println(ui.ErrorStyle.Render(fmt.Sprintf("Error: %v", err)))
		return ""
	}

	repo := fmt.Sprintf("%s/%s", username, name)

	fmt.Println()
	fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Creating private repository %s...", repo)))

	cmd := exec.Command("gh", "repo", "create", name, "--private", "--description", "Vaulty secrets repository", "--confirm")
	output, err := cmd.CombinedOutput()
	if err != nil {

		if strings.Contains(string(output), "already exists") {
			fmt.Println(ui.WarningStyle.Render(fmt.Sprintf("Repository %s already exists", repo)))
			return repo
		}
		fmt.Println(ui.ErrorStyle.Render(fmt.Sprintf("Failed to create repository: %s", string(output))))
		return ""
	}

	time.Sleep(2 * time.Second)

	fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("✓ Repository %s created", repo)))
	return repo
}

func linkExistingRepo() string {
	fmt.Println()
	fmt.Println(ui.InfoStyle.Render("🔗 Linking an existing repository..."))
	fmt.Println()

	var url string
	for {
		err := huh.NewInput().
			Title("Repository URL or owner/repo").
			Placeholder("https://github.com/owner/repo or owner/repo").
			Value(&url).
			Validate(func(s string) error {
				if s == "" {
					return fmt.Errorf("URL is required")
				}
				return nil
			}).
			Run()
		if err != nil {
			return ""
		}

		owner, repo, err := parseRepoFromInput(url)
		if err != nil {
			fmt.Println(ui.ErrorStyle.Render(fmt.Sprintf("Invalid format: %v", err)))
			continue
		}

		repoFull := fmt.Sprintf("%s/%s", owner, repo)

		fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Verifying repository %s...", repoFull)))

		token, err := github.GetGitHubToken()
		if err != nil {
			fmt.Println(ui.ErrorStyle.Render(fmt.Sprintf("GitHub authentication: %v", err)))
			continue
		}

		client := github.NewClient(token)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_, err = client.ListDirectory(ctx, owner, repo, "")
		if err != nil {
			fmt.Println(ui.ErrorStyle.Render(fmt.Sprintf("Cannot access repository: %v", err)))
			continue
		}

		fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("✓ Repository verified: %s", repoFull)))
		return repoFull
	}
}

func getGitHubUsername() (string, error) {
	cmd := exec.Command("gh", "api", "user", "-q", ".login")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("getting GitHub username: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

func parseRepoFromInput(input string) (owner, repo string, err error) {
	input = strings.TrimSpace(input)

	if strings.Contains(input, "github.com") {

		re := regexp.MustCompile(`github\.com[/:]([^/]+)/([^/]+?)(?:\.git)?$`)
		matches := re.FindStringSubmatch(input)
		if len(matches) == 3 {
			return matches[1], strings.TrimSuffix(matches[2], ".git"), nil
		}
		return "", "", fmt.Errorf("could not parse GitHub URL")
	}

	return github.ParseRepo(input)
}

func isValidRepoName(name string) bool {
	if name == "" {
		return false
	}

	if len(name) > 100 {
		return false
	}
	if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "-") {
		return false
	}
	re := regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
	return re.MatchString(name)
}

func init() {
	rootCmd.AddCommand(initCmd)
}
