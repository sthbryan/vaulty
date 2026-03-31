package main

import (
	"github.com/spf13/cobra"
)

var exportOutput string

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export vault to a backup file",
	Long: `Export all secrets from the vault to a single encrypted backup file.

The backup is encrypted with a password you provide and can be restored
using 'vty import'.

Examples:
  vty export
  vty export -o my-backup.vtyb`,
	Args: cobra.NoArgs,
	RunE: runExport,
}

func init() {
	rootCmd.AddCommand(exportCmd)
	exportCmd.Flags().StringVarP(&exportOutput, "output", "o", "vaulty-backup.vtyb", "Output file path")
}
