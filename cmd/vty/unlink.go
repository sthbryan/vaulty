package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/DeadBryam/vaulty/internal/config"
	"github.com/DeadBryam/vaulty/internal/password"
	"github.com/DeadBryam/vaulty/internal/session"
	"github.com/DeadBryam/vaulty/internal/ui"
	"github.com/spf13/cobra"
)

var unlinkForce bool

var unlinkCmd = &cobra.Command{
	Use:   "unlink",
	Short: "Unlink Vaulty from the current repository",
	Long: `Unlink Vaulty by removing the local configuration file.

⚠️  WARNING: This will delete your local configuration at ~/.vty/config.json.
This does NOT delete any secrets from your GitHub repository.

Your encrypted secrets will remain safe in the repository.
You can re-link Vaulty anytime by running 'vty init' again.`,
	RunE: runUnlink,
}

func runUnlink(cmd *cobra.Command, args []string) error {
	configPath := config.DefaultPath()

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Println()
		logger.Info("Vaulty is not linked to any repository")
		logger.Info("Run 'vty init' to link Vaulty to a repository")
		return nil
	}

	sessionMgr := session.GetManager()
	activeUsers := sessionMgr.All()
	if len(activeUsers) > 0 {
		logger.Info("Clearing active sessions...")
		sessionMgr.Clear()
	}

	fmt.Println()
	logger.Warn("⚠️  WARNING: This will unlink Vaulty completely!")
	logger.Warn("   Local data will be removed:")
	logger.Warn(fmt.Sprintf("   - Configuration: %s", configPath))
	home, _ := os.UserHomeDir()
	cacheDir := filepath.Join(home, ".vty", "cache")
	logger.Warn(fmt.Sprintf("   - Cache directory: %s", cacheDir))
	logger.Warn("   - Stored password")
	fmt.Println()
	logger.Info("   Your encrypted secrets in GitHub will NOT be affected.")
	fmt.Println()

	if !unlinkForce {
		confirmed, err := ui.AskConfirm("   Are you sure you want to unlink?", false)
		if err != nil {
			return fmt.Errorf("failed to get confirmation: %w", err)
		}
		if !confirmed {
			fmt.Println()
			logger.Info("Unlink cancelled")
			return nil
		}
	}

	if err := os.Remove(configPath); err != nil {
		logger.Error("Failed to delete config file", "error", err)
		return fmt.Errorf("deleting config: %w", err)
	}

	sessionMgr.Clear()

	home, err := os.UserHomeDir()
	if err == nil {
		vautyDir := filepath.Join(home, ".vaulty")
		if err := os.RemoveAll(vautyDir); err != nil && !os.IsNotExist(err) {
			logger.Warn("Failed to delete .vaulty directory", "error", err)
		}
	}

	if err == nil {
		cacheDir := filepath.Join(home, ".vty", "cache")
		if err := os.RemoveAll(cacheDir); err != nil && !os.IsNotExist(err) {
			logger.Warn("Failed to delete cache directory", "error", err)
		}
	}

	passStorage, err := password.NewStorage()
	if err == nil {
		if err := passStorage.Delete(); err != nil {
			logger.Warn("Failed to delete stored password", "error", err)
		}
	}

	fmt.Println()
	logger.Info("✅ Vaulty unlinked. All local data removed. GitHub vault untouched.")
	fmt.Println()

	return nil
}

func init() {
	unlinkCmd.Flags().BoolVarP(&unlinkForce, "force", "f", false, "Force unlink without confirmation")
	rootCmd.AddCommand(unlinkCmd)
}
