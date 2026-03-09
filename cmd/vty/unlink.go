package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/sthbryan/vaulty/internal/config"
	"github.com/sthbryan/vaulty/internal/ui"
)

var unlinkForce bool

var unlinkCmd = &cobra.Command{
	Use:   "unlink",
	Short: "⚠️  Unlink Vaulty from the current repository",
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

	fmt.Println()
	logger.Warn("⚠️  WARNING: You are about to unlink Vaulty!")
	logger.Warn("   This will delete your local configuration file:")
	logger.Warn(fmt.Sprintf("   %s", configPath))
	fmt.Println()
	logger.Info("   Your encrypted secrets in GitHub will NOT be affected.")
	fmt.Println()

	if !unlinkForce {
		confirmed, err := ui.AskConfirm("   Are you sure you want to unlink Vaulty?", false)
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

	fmt.Println()
	logger.Info("✅ Vaulty has been unlinked successfully!")
	logger.Info(fmt.Sprintf("   Config deleted: %s", configPath))
	fmt.Println()
	logger.Info("Your secrets remain safe in GitHub.")
	logger.Info("Run 'vty init' to link Vaulty again.")

	return nil
}

func init() {
	unlinkCmd.Flags().BoolVarP(&unlinkForce, "force", "f", false, "Force unlink without confirmation")
	rootCmd.AddCommand(unlinkCmd)
}
