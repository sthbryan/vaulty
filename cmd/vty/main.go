package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "vty",
	Short: "Vaulty - Secret management CLI",
	Long:  `A secure CLI for managing secrets, environment variables, and configurations.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Vaulty - Secret management CLI")
		fmt.Println("Run 'vty --help' for available commands")
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}