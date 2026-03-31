package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/DeadBryam/vaulty/internal/cli"
	"github.com/DeadBryam/vaulty/internal/compress"
	"github.com/DeadBryam/vaulty/internal/crypto"
	"github.com/spf13/cobra"
)

func runShowEnv(cmd *cobra.Command, args []string) error {
	name := args[0]
	if err := cli.ValidateName(name); err != nil {
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

	if showEnv != "" {
		if !cfg.HasEnvironment(showEnv) {
			return fmt.Errorf("environment %q not defined. Defined: %v", showEnv, cfg.GetEnvironments())
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var env string
	if showEnv != "" {
		env = showEnv
	}

	logger.Info("Downloading secrets...", "name", name)

	encodedData, err := s.GetEnv(ctx, env, name)
	if err != nil {
		return fmt.Errorf("secret not found: %s", name)
	}

	return displayContent(encodedData, sess.MasterKey, name, "env")
}

func runShowSSH(cmd *cobra.Command, args []string) error {
	name := args[0]
	if err := cli.ValidateName(name); err != nil {
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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logger.Info("Downloading SSH key...", "name", name)

	data, err := s.GetSSHKey(ctx, sess.Username, name)
	if err != nil {
		return fmt.Errorf("SSH key not found: %s", name)
	}

	hexData := string(data)
	vaultJSON, err := crypto.DecryptBinary(hexData, sess.MasterKey)
	if err != nil {
		if err == crypto.ErrDecryptionFailed {
			return fmt.Errorf("decryption failed: invalid password")
		}
		return fmt.Errorf("decrypting: %w", err)
	}

	var vaultFile BinaryVaultFile
	if err := json.Unmarshal(vaultJSON, &vaultFile); err != nil {
		return fmt.Errorf("parsing vault file: %w", err)
	}

	content, err := compress.Decompress(vaultFile.Data)
	if err != nil {
		return fmt.Errorf("decompressing: %w", err)
	}

	return displaySSHPreview(content, name)
}

func runShowResource(cmd *cobra.Command, args []string) error {
	name := args[0]
	if err := cli.ValidateName(name); err != nil {
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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logger.Info("Downloading resource...", "name", name)

	path := "resources/" + name
	if !strings.HasSuffix(name, ".vty") {
		path += ".vty"
	}

	content, err := s.GetResource(ctx, path)
	if err != nil {

		content, err = s.GetResource(ctx, name)
		if err != nil {
			return fmt.Errorf("resource not found: %s", name)
		}
	}

	return displayContent(content, sess.MasterKey, name, "resource")
}

func runShowConfig(cmd *cobra.Command, args []string) error {
	name := args[0]
	if err := cli.ValidateName(name); err != nil {
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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logger.Info("Downloading config...", "name", name)

	path := "config/" + name
	if !strings.HasSuffix(name, ".vty") {
		path += ".vty"
	}

	content, err := s.GetResource(ctx, path)
	if err != nil {
		return fmt.Errorf("config not found: %s", name)
	}

	return displayContent(content, sess.MasterKey, name, "config")
}

func displayContent(encodedData []byte, masterKey []byte, name, secretType string) error {
	hexData := string(encodedData)
	vaultJSON, err := crypto.DecryptBinary(hexData, masterKey)
	if err != nil {
		if err == crypto.ErrDecryptionFailed {
			return fmt.Errorf("decryption failed: invalid password or corrupted data")
		}
		return fmt.Errorf("decrypting: %w", err)
	}

	var vaultFile BinaryVaultFile
	if err := json.Unmarshal(vaultJSON, &vaultFile); err != nil {
		return fmt.Errorf("parsing vault file: %w", err)
	}

	content, err := compress.Decompress(vaultFile.Data)
	if err != nil {
		return fmt.Errorf("decompressing: %w", err)
	}

	return printWithPager(content, name, secretType)
}

func displaySSHPreview(content []byte, name string) error {
	lines := strings.Split(string(content), "\n")

	fmt.Printf("=== SSH Key: %s ===\n\n", name)

	fingerprint, err := getSSHFingerprint(content)
	if err == nil {
		fmt.Printf("Fingerprint: %s\n\n", fingerprint)
	}

	fmt.Println("Preview (first 2 lines):")
	for i := 0; i < 2 && i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) != "" {
			fmt.Printf("  %s\n", lines[i])
		}
	}

	fmt.Println("\n[...]")

	fmt.Println("\nPreview (last 2 lines):")
	start := len(lines) - 2
	if start < 0 {
		start = 0
	}
	for i := start; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) != "" {
			fmt.Printf("  %s\n", lines[i])
		}
	}

	fmt.Println("\n(Use 'vty pull ssh' to download the full key)")

	return nil
}

func getSSHFingerprint(content []byte) (string, error) {

	cmd := exec.Command("ssh-keygen", "-lf", "-")
	cmd.Stdin = bytes.NewReader(content)

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = bytes.NewBuffer(nil)

	if err := cmd.Run(); err != nil {
		return "", err
	}

	return strings.TrimSpace(out.String()), nil
}

func printWithPager(content []byte, name, secretType string) error {

	if _, err := exec.LookPath("bat"); err == nil {

		f, err := os.CreateTemp("", "vty-*."+secretType)
		if err == nil {
			defer os.Remove(f.Name())
			f.Write(content)
			f.Close()

			cmd := exec.Command("bat", "--style=header,grid", "--language=env", f.Name())
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			return cmd.Run()
		}
	}

	_, err := os.Stdout.Write(content)
	return err
}
