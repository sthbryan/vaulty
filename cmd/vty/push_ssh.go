package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/sthbryan/vaulty/internal/ui"
	"github.com/sthbryan/vaulty/pkg/models"
)

func runPushSSH(cmd *cobra.Command, args []string) error {
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

	if err := validateFile(path); err != nil {
		return err
	}

	vaultFile, originalSize, err := encryptAndPrepareFileWithSession(path, name, models.SecretTypeSSH)
	if err != nil {
		return err
	}

	remotePath := fmt.Sprintf("ssh/%s/%s.vty", sess.Username, name)

	if !s.IsLocal() {
		ui.PrintCloud("Ensuring SSH directory exists for user: %s", sess.Username)
	}

	encryptedSize, err := encryptAndUploadWithStorage(s, remotePath, vaultFile, sess.MasterKey)
	if err != nil {
		return err
	}

	ui.PrintSuccess("Pushed SSH key successfully!")
	fmt.Println()
	fmt.Printf("  Name:    %s\n", name)
	fmt.Printf("  User:    %s\n", sess.Username)
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
