package main

import (
	"github.com/spf13/cobra"
)

var importInput string

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import vault from a backup file",
	Long: `Import secrets from a backup file created with 'vty export'.

The backup file is decrypted using the vault password, and secrets are
imported to the current vault.

Examples:
  vty import
  vty import -i my-backup.vtyb`,
	Args: cobra.NoArgs,
	RunE: runImport,
}

func init() {
	rootCmd.AddCommand(importCmd)
	importCmd.Flags().StringVarP(&importInput, "input", "i", "vaulty-backup.vtyb", "Input file path")
}
