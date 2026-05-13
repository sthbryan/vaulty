package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/sthbryan/vaulty/v2/internal/ui"
)

var rootCmd = &cobra.Command{
	Use:           "vty",
	Short:         "Vaulty - Secret management CLI",
	Long:          "A secure CLI for managing secrets, environment variables, configurations and more.",
	SilenceUsage:  true,
	SilenceErrors: true,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		ui.PrintError(fmt.Sprintf("Error: %v", err))
		os.Exit(1)
	}
}
