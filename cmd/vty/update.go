package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	updateCmd = &cobra.Command{
		Use:   "update",
		Short: "Check for updates and upgrade Vaulty",
		Long: `Check if a new version of Vaulty is available and upgrade to it.

Examples:
  vty update          # Check for updates and prompt to upgrade
  vty update --force  # Force upgrade without confirmation
  vty update --check  # Only check, don't upgrade`,
		RunE: runUpdate,
	}

	updateForce bool
	updateCheck bool
)

const (
	owner       = "sthbryan"
	repo        = "vaulty"
	latestURL   = "https://api.github.com/repos/" + owner + "/" + repo + "/releases/latest"
	currentRepo = "https://github.com/" + owner + "/" + repo
)

var (
	infoStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#04B575"))
	warnStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFA500"))
	successStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#04B575"))
)

func runUpdate(cmd *cobra.Command, args []string) error {
	if updateCheck {
		return checkForUpdate(false)
	}
	return checkForUpdate(!updateForce)
}

func checkForUpdate(askConfirmation bool) error {
	fmt.Println()
	logger.Info("Checking for updates...")

	currentVersion := strings.TrimPrefix(version, "v")
	if currentVersion == "" {
		currentVersion = "unknown"
	}

	latestVersion, downloadURL, err := getLatestRelease()
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	fmt.Println(infoStyle.Render("Current version:  ") + version)
	fmt.Println(infoStyle.Render("Latest version:   ") + "v" + latestVersion)

	if latestVersion == currentVersion || currentVersion == "unknown" {
		fmt.Println()
		successStyle.Render("✓ You're running the latest version!")
		fmt.Println(successStyle.Render("You're already running the latest version of Vaulty!"))
		return nil
	}

	fmt.Println()
	fmt.Println(warnStyle.Render("A new version is available!"))
	fmt.Printf("Please upgrade from %s to v%s\n", version, latestVersion)
	fmt.Println()

	if !askConfirmation {
		fmt.Println("Run 'vty update --force' to upgrade automatically.")
		fmt.Println("Or visit " + currentRepo + "/releases for manual download.")
		return nil
	}

	if !confirmUpgrade() {
		fmt.Println("Update cancelled.")
		return nil
	}

	return downloadAndInstall(downloadURL, latestVersion)
}

func getLatestRelease() (string, string, error) {
	client := &http.Client{}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, latestURL, nil)
	if err != nil {
		return "", "", err
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "Vaulty-Update-Checker")

	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return "", "", fmt.Errorf("repository not found or no releases available")
		}
		return "", "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	tagName := extractJSONValue(string(body), "tag_name")
	downloadURL := getDownloadURL(string(body))

	if tagName == "" {
		return "", "", fmt.Errorf("could not parse release information")
	}

	version := strings.TrimPrefix(tagName, "v")
	return version, downloadURL, nil
}

func extractJSONValue(json, key string) string {
	start := fmt.Sprintf(`"%s"`, key)
	idx := strings.Index(json, start)
	if idx == -1 {
		return ""
	}

	colonIdx := idx + len(start) + 1
	for colonIdx < len(json) && (json[colonIdx] == ' ' || json[colonIdx] == ':') {
		colonIdx++
	}

	if colonIdx >= len(json) {
		return ""
	}

	if json[colonIdx] == '"' {
		colonIdx++
		end := colonIdx
		for end < len(json) && json[end] != '"' {
			if json[end] == '\\' && end+1 < len(json) {
				end++
			}
			end++
		}
		return json[colonIdx:end]
	}

	end := colonIdx
	for end < len(json) && json[end] != ',' && json[end] != '}' && json[end] != ' ' {
		end++
	}
	return strings.TrimSpace(json[colonIdx:end])
}

func getDownloadURL(releaseJSON string) string {
	assetsIdx := strings.Index(releaseJSON, `"assets"`)
	if assetsIdx == -1 {
		return ""
	}

	browserDownloadIdx := strings.Index(releaseJSON[assetsIdx:], `"browser_download_url"`)
	if browserDownloadIdx == -1 {
		return ""
	}

	urlStart := assetsIdx + browserDownloadIdx + len(`"browser_download_url"`) + 2
	urlEnd := urlStart

	for urlEnd < len(releaseJSON) && releaseJSON[urlEnd] != '"' {
		if releaseJSON[urlEnd] == '\\' && urlEnd+1 < len(releaseJSON) {
			urlEnd++
		}
		urlEnd++
	}

	return releaseJSON[urlStart:urlEnd]
}

func confirmUpgrade() bool {
	fmt.Print("Do you want to upgrade now? [Y/n]: ")
	var input string
	fmt.Scanln(&input)

	input = strings.TrimSpace(strings.ToLower(input))
	return input == "" || input == "y" || input == "yes"
}

func downloadAndInstall(downloadURL, version string) error {
	fmt.Println()
	logger.Info("Downloading v" + version + "...")

	tmpDir, err := os.MkdirTemp("", "vaulty-update-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	binaryName := getBinaryName()
	tmpPath := tmpDir + "/" + binaryName

	if err := downloadFile(downloadURL, tmpPath); err != nil {
		return fmt.Errorf("failed to download update: %w", err)
	}

	if err := os.Chmod(tmpPath, 0755); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	currentBin, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to find current executable: %w", err)
	}

	fmt.Println()
	logger.Info("Installing update...")

	if err := atomicReplace(currentBin, tmpPath); err != nil {
		return fmt.Errorf("failed to install update: %w", err)
	}

	fmt.Println()
	successStyle.Render("✓ Update installed successfully!")
	fmt.Println(successStyle.Render("Upgrade complete! Run 'vty --version' to verify."))
	return nil
}

func getBinaryName() string {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	base := "vty"

	switch goos {
	case "darwin":
		base += "-darwin-" + goarch
	case "linux":
		base += "-linux-" + goarch
	case "windows":
		base += "-windows-amd64.exe"
	default:
		base += "-" + goos + "-" + goarch
	}

	return base
}

func downloadFile(url, path string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func atomicReplace(target, tmpPath string) error {
	switch runtime.GOOS {
	case "darwin", "linux":
		return atomicReplaceUnix(target, tmpPath)
	case "windows":
		return atomicReplaceWindows(target, tmpPath)
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

func atomicReplaceUnix(target, tmpPath string) error {
	if err := os.Rename(tmpPath, target); err != nil {
		cmd := exec.Command("cp", tmpPath, target)
		if err := cmd.Run(); err != nil {
			return err
		}
	}
	return nil
}

func atomicReplaceWindows(target, tmpPath string) error {
	batContent := fmt.Sprintf(`@echo off
copy /Y "%s" "%s"
del "%s"`, tmpPath, target, tmpPath)

	batPath := os.TempDir() + "/vaulty-upgrade.bat"
	if err := os.WriteFile(batPath, []byte(batContent), 0644); err != nil {
		return err
	}

	cmd := exec.Command("cmd", "/c", "start", "/wait", batPath)
	if err := cmd.Run(); err != nil {
		return err
	}

	os.Remove(batPath)
	return nil
}

func init() {
	rootCmd.AddCommand(updateCmd)
	updateCmd.Flags().BoolVar(&updateForce, "force", false, "Force upgrade without confirmation")
	updateCmd.Flags().BoolVar(&updateCheck, "check", false, "Only check for updates, don't upgrade")
}
