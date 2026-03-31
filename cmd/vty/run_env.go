package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/DeadBryam/vaulty/internal/cli"
	"github.com/DeadBryam/vaulty/internal/compress"
	"github.com/DeadBryam/vaulty/internal/config"
	"github.com/DeadBryam/vaulty/internal/crypto"
	"github.com/spf13/cobra"
)

func runRunEnv(cmd *cobra.Command, args []string) error {
	sepIndex := -1
	for i, arg := range args {
		if arg == "--" {
			sepIndex = i
			break
		}
	}

	if sepIndex == -1 {
		return fmt.Errorf("missing '--' separator. Usage: vty run env <name> [--env <env>] -- <command> [args...]")
	}

	if sepIndex == 0 || sepIndex >= len(args)-1 {
		return fmt.Errorf("missing command after '--'. Usage: vty run env <name> [--env <env>] -- <command> [args...]")
	}

	name := args[0]
	if err := cli.ValidateName(name); err != nil {
		return err
	}

	commandArgs := args[sepIndex+1:]
	if len(commandArgs) == 0 {
		return fmt.Errorf("missing command after '--'. Usage: vty run env <name> [--env <env>] -- <command> [args...]")
	}

	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	sess, err := ensureAuthenticated(cfg)
	if err != nil {
		return err
	}

	if runEnv != "" {
		if !cfg.HasEnvironment(runEnv) {
			return fmt.Errorf("environment %q not defined. Defined: %v", runEnv, cfg.GetEnvironments())
		}
	}

	s, err := getStorage(cfg)
	if err != nil {
		return fmt.Errorf("failed to get storage: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var env, envName string
	var remotePath string

	if runEnv != "" {
		env = runEnv
		envName = name
		remotePath = fmt.Sprintf("envs/%s/%s.vty", runEnv, name)
	} else {
		env = ""
		envName = name
		remotePath = fmt.Sprintf("envs/%s.vty", name)
	}

	logger.Info("Downloading secrets...", "name", name)

	encodedData, err := s.GetEnv(ctx, env, envName)
	if err != nil {
		return fmt.Errorf("secret not found: %s", remotePath)
	}

	logger.Info("Downloaded", "path", remotePath)

	logger.Info("Decrypting...")
	hexData := string(encodedData)
	vaultJSON, err := crypto.DecryptBinary(hexData, sess.MasterKey)
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

	plaintext, err := compress.Decompress(vaultFile.Data)
	if err != nil {
		return fmt.Errorf("decompressing data: %w", err)
	}

	envVars, err := parseEnvContent(string(plaintext))
	if err != nil {
		return fmt.Errorf("parsing .env: %w", err)
	}

	mergedEnv := os.Environ()
	for key, value := range envVars {
		mergedEnv = append(mergedEnv, fmt.Sprintf("%s=%s", key, value))
	}

	cmdExec := exec.Command(commandArgs[0], commandArgs[1:]...)
	cmdExec.Env = mergedEnv
	cmdExec.Stdout = os.Stdout
	cmdExec.Stderr = os.Stderr
	cmdExec.Stdin = os.Stdin

	logger.Info("Executing command...", "command", strings.Join(commandArgs, " "))

	if err := cmdExec.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return fmt.Errorf("command failed: %w", err)
	}

	return nil
}

func parseEnvContent(content string) (map[string]string, error) {
	result := make(map[string]string)
	lines := strings.Split(content, "\n")

	for lineNum, line := range lines {
		line = strings.TrimRight(line, "\r")

		if strings.TrimSpace(line) == "" {
			continue
		}

		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}

		eqIndex := strings.Index(line, "=")
		if eqIndex == -1 {
			return nil, fmt.Errorf("invalid .env format at line %d: %q (missing '=')", lineNum+1, line)
		}

		key := strings.TrimSpace(line[:eqIndex])
		value := line[eqIndex+1:]

		if strings.HasPrefix(key, "export ") {
			key = strings.TrimPrefix(key, "export ")
			key = strings.TrimSpace(key)
		}

		if key == "" {
			return nil, fmt.Errorf("invalid .env format at line %d: %q (empty key)", lineNum+1, line)
		}

		value = strings.Trim(value, " \t")
		if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
			value = strings.Trim(value, "\"")
		} else if strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") {
			value = strings.Trim(value, "'")
		}

		result[key] = value
	}

	return result, nil
}
