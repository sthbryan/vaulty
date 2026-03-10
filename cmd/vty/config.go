package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/sthbryan/vaulty/internal/config"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure Vaulty settings",
}

var cacheDurationCmd = &cobra.Command{
	Use:   "cache-duration [duration]",
	Short: "Get or set password cache duration",
	RunE:  runCacheDuration,
}

func runCacheDuration(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if len(args) == 0 {
		if cfg.CacheDuration == "" {
			fmt.Println("15m")
		} else {
			fmt.Println(cfg.CacheDuration)
		}
		return nil
	}

	duration := args[0]
	d, err := time.ParseDuration(duration)
	if err != nil {
		return fmt.Errorf("invalid duration: %w", err)
	}

	minDuration := time.Minute
	maxDuration := 24 * time.Hour

	if d < minDuration {
		return fmt.Errorf("duration must be at least 1m")
	}
	if d > maxDuration {
		return fmt.Errorf("duration must be at most 24h")
	}

	cfg.CacheDuration = duration
	if err := cfg.Save(""); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Printf("Cache duration set to %s\n", duration)
	return nil
}

func init() {
	configCmd.AddCommand(cacheDurationCmd)
	rootCmd.AddCommand(configCmd)
}
