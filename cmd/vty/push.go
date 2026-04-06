package main

import (
	"github.com/spf13/cobra"
)

var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Store secrets in vault",
	Long:  `Store secrets in your vault.`,
}

func init() {
	rootCmd.AddCommand(pushCmd)
}
