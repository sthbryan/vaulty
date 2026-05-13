package main

import (
	"github.com/spf13/cobra"
)

var (
	pullOutput      string
	pullInteractive bool
	pullUser        string
	pullEnv         string
)

var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Get secrets from vault",
	Long:  `Get secrets from your vault.`,
}

func init() {
	rootCmd.AddCommand(pullCmd)
}
