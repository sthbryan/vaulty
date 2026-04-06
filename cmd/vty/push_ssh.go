package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/sthbryan/vaulty/internal/ui"
	"github.com/sthbryan/vaulty/pkg/models"
)

var pushSSHCmd = &cobra.Command{
	Use:   "ssh <name> <path>",
	Short: "Push an SSH key to Vaulty",
	Long: `Compress, encrypt, and upload an SSH private key to your Vaulty repository.


The file will be:
  1. Compressed using gzip for efficiency
  2. Encrypted using AES-256-GCM with your password
  3. Uploaded to ssh/{username}/{name}.vty in your repository

Only owners and editors can push SSH keys to their own directory.
Viewers cannot push any secrets.

Examples:
  vty push ssh laptop ~/.ssh/id_rsa
  vty push ssh server ~/.ssh/server_key --force`,
	Args: cobra.ExactArgs(2),
	RunE: runPushSSH,
}

func init() {
	pushCmd.AddCommand(pushSSHCmd)
	pushSSHCmd.Flags().BoolVarP(&pushForce, "force", "f", false, "Overwrite without prompting")
}

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
