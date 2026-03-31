package main

import (
	"context"
	"fmt"
	"time"

	"github.com/DeadBryam/vaulty/internal/ui"
	"github.com/spf13/cobra"
)

var deleteResourceTag string

var deleteResourceCmd = &cobra.Command{
	Use:   "resource <name>",
	Short: "Delete a resource from the vault",
	Long: `Delete a file or directory from the resources/ directory.

Examples:
  vty delete resource agents
  vty delete resource zellij --tag dev`,
	Args: cobra.ExactArgs(1),
	RunE: runDeleteResource,
}

var deleteConfigCmd = &cobra.Command{
	Use:   "config <name>",
	Short: "Delete a config from the vault",
	Long: `Delete a file or directory from the config/ directory.

Examples:
  vty delete config opencode
  vty delete config zellij --tag team`,
	Args: cobra.ExactArgs(1),
	RunE: runDeleteConfig,
}

func runDeleteResource(cmd *cobra.Command, args []string) error {
	return runDeleteResourceOrConfig(args[0], "resources")
}

func runDeleteConfig(cmd *cobra.Command, args []string) error {
	return runDeleteResourceOrConfig(args[0], "config")
}

func runDeleteResourceOrConfig(name, baseDir string) error {
	if err := validateName(name); err != nil {
		return err
	}
	if deleteResourceTag != "" {
		if err := validateName(deleteResourceTag); err != nil {
			return fmt.Errorf("invalid tag: %w", err)
		}
	}

	s, cfg, err := getStorageForDelete()
	if err != nil {
		return err
	}

	sess, err := ensureAuthenticated(cfg)
	if err != nil {
		return err
	}

	if err := checkPushPermissions(sess.Role); err != nil {
		return err
	}

	var remotePath string
	if deleteResourceTag != "" {
		remotePath = fmt.Sprintf("%s/%s/%s.vty", baseDir, deleteResourceTag, name)
	} else {
		remotePath = fmt.Sprintf("%s/%s.vty", baseDir, name)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err = s.GetResource(ctx, remotePath)
	if err != nil {
		return fmt.Errorf("resource not found: %s (try with --tag flag)", remotePath)
	}

	if !deleteForce {
		fmt.Printf("Delete %s from %s? ", name, remotePath)
		confirmed, err := ui.AskConfirm("", false)
		if err != nil || !confirmed {
			fmt.Println("Cancelled")
			return nil
		}
	}

	err = s.DeleteResource(ctx, remotePath)
	if err != nil {
		return fmt.Errorf("failed to delete: %w", err)
	}

	fmt.Println(ui.SuccessStyle.Render("✅ Deleted: " + name))

	return nil
}

func init() {
	deleteResourceCmd.Flags().StringVarP(&deleteResourceTag, "tag", "t", "", "Tag of the resource (e.g., dev, team)")
	deleteResourceCmd.Flags().BoolVarP(&deleteForce, "force", "f", false, "Delete without prompting")

	deleteConfigCmd.Flags().StringVarP(&deleteResourceTag, "tag", "t", "", "Tag of the config (e.g., dev, team)")
	deleteConfigCmd.Flags().BoolVarP(&deleteForce, "force", "f", false, "Delete without prompting")

	deleteCmd.AddCommand(deleteResourceCmd)
	deleteCmd.AddCommand(deleteConfigCmd)
}
