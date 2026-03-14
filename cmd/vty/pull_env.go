package main

import (
	"context"
	"fmt"
	"time"

	"github.com/DeadBryam/vaulty/internal/config"
	"github.com/DeadBryam/vaulty/internal/session"
	"github.com/DeadBryam/vaulty/internal/storage"
	"github.com/DeadBryam/vaulty/internal/ui"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

func runPullEnv(cmd *cobra.Command, args []string) error {
	name := args[0]

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

	remotePath, err := getRemotePathForEnv(name, pullEnv, cfg, sess)
	if err != nil {
		return err
	}

	return pullSecretWithRemotePath(name, remotePath, sess)
}

func getRemotePathForEnv(name, envFlag string, cfg *config.Config, sess *session.Session) (string, error) {

	environments := cfg.Environments
	if len(environments) == 0 {
		environments = []string{"production"}
	}

	if envFlag != "" {

		if envFlag == "all" {
			return selectEnvironmentAndBuildPath(name, environments)
		}

		validEnv := false
		for _, e := range environments {
			if e == envFlag {
				validEnv = true
				break
			}
		}
		if !validEnv {
			return "", fmt.Errorf("unknown environment: %s (available: %v)", envFlag, environments)
		}

		return fmt.Sprintf("envs/%s/%s.vty", envFlag, name), nil
	}

	sharedPath := fmt.Sprintf("envs/%s.vty", name)

	s, err := getStorage(cfg)
	var _ storage.Storage = s
	if err != nil {
		return "", fmt.Errorf("failed to get storage: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = s.GetEnv(ctx, "", name)
	if err == nil {

		return sharedPath, nil
	}

	logger.Info("No shared secrets found, showing environment selector...")
	return selectEnvironmentAndBuildPath(name, environments)
}

func selectEnvironmentAndBuildPath(name string, environments []string) (string, error) {
	if !pullInteractive {
		return "", fmt.Errorf("secret not found in shared location and no environment specified (use --env or -e flag, or -i for interactive mode)")
	}

	fmt.Println()
	fmt.Println(ui.InfoStyle.Render("🔍 Select environment:"))

	var selectedEnv string
	var options []huh.Option[string]

	for _, env := range environments {
		options = append(options, huh.NewOption(env, env))
	}

	err := huh.NewSelect[string]().
		Title("Choose an environment").
		Options(options...).
		Value(&selectedEnv).
		Run()
	if err != nil {
		return "", fmt.Errorf("selection cancelled")
	}

	return fmt.Sprintf("envs/%s/%s.vty", selectedEnv, name), nil
}
