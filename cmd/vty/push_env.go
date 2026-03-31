package main

import (
	"fmt"

	"github.com/DeadBryam/vaulty/internal/ui"
	"github.com/DeadBryam/vaulty/pkg/models"
	"github.com/spf13/cobra"
)

func runPushEnv(cmd *cobra.Command, args []string) error {
	name := args[0]
	path := args[1]

	if err := validateName(name); err != nil {
		return err
	}

	cfg, s, err := loadConfigAndStorage()
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

	if pushEnv != "" && pushEnv != "all" {
		if !cfg.HasEnvironment(pushEnv) {
			return fmt.Errorf("environment %q not defined in config. Defined: %v", pushEnv, cfg.GetEnvironments())
		}
	}

	if err := validateFile(path); err != nil {
		return err
	}

	vaultFile, originalSize, err := encryptAndPrepareFileWithSession(path, name, models.SecretTypeEnv)
	if err != nil {
		return err
	}

	var remotePath string
	switch pushEnv {
	case "":
		remotePath = fmt.Sprintf("envs/%s.vty", name)
	case "all":

		confirmed, confirmErr := ui.AskConfirm(fmt.Sprintf("Push %s to all environments?", name), false)
		if confirmErr != nil {
			return fmt.Errorf("confirmation failed: %w", confirmErr)
		}
		if !confirmed {
			ui.PrintInfo("Push cancelled")
			return nil
		}

		envs := cfg.GetEnvironments()
		for _, env := range envs {
			envPath := fmt.Sprintf("envs/%s/%s.vty", env, name)
			ui.PrintInfo("Pushing to environment: %s", env)
			if _, err := encryptAndUploadWithStorage(s, envPath, vaultFile, sess.MasterKey); err != nil {
				return fmt.Errorf("failed to push to %s: %w", env, err)
			}
		}
		ui.PrintSuccess("Pushed to all environments successfully!")
		fmt.Println()
		fmt.Printf("  Name:    %s\n", name)
		fmt.Printf("  Envs:    %v\n", envs)
		fmt.Printf("  Size:    %s → %s\n",
			ui.FormatBytes(originalSize),
			ui.FormatBytes(int64(len(vaultFile.Data)*2)))
		if cfg.IsLocalMode() {
			fmt.Printf("  Storage: local (%s)\n", s.GetRepo())
		} else {
			fmt.Printf("  Repo:    %s\n", cfg.Repo)
		}
		return nil
	default:
		remotePath = fmt.Sprintf("envs/%s/%s.vty", pushEnv, name)
	}

	encryptedSize, err := encryptAndUploadWithStorage(s, remotePath, vaultFile, sess.MasterKey)
	if err != nil {
		return err
	}

	ui.PrintSuccess("Pushed successfully!")
	fmt.Println()
	fmt.Printf("  Name:    %s\n", name)
	fmt.Printf("  Path:    %s\n", remotePath)
	fmt.Printf("  Size:    %s → %s\n",
		ui.FormatBytes(originalSize),
		ui.FormatBytes(int64(encryptedSize)))
	if cfg.IsLocalMode() {
		fmt.Printf("  Storage: local (%s)\n", s.GetRepo())
	} else {
		fmt.Printf("  Repo:    %s\n", cfg.Repo)
	}

	return nil
}
