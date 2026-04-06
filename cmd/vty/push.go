package main

import (
	"github.com/spf13/cobra"
)

var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push secrets to Vaulty",
	Long:  `Push environment files or SSH keys to your Vaulty repository.`,
}

func init() {
	rootCmd.AddCommand(pushCmd)
}
