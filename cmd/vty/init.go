package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/sthbryan/vaulty/internal/config"
	"github.com/sthbryan/vaulty/internal/ui"
)

var localMode bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Vaulty with a GitHub repository or local storage",
	Long: `Initialize Vaulty by creating a new vault.

This command will guide you through:
  • Setting up a secure master password
  • Choosing storage mode (GitHub or local)

Use --local for local-only storage without GitHub.`,
	RunE: runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	fmt.Println()
	ui.PrintAnimatedLogo()

	if localMode {
		fmt.Println(ui.TitleStyle.Render("✨ Welcome to Vaulty (Local Mode)!"))
		fmt.Println(ui.MutedStyle.Render("  Secure secret management on your local filesystem"))
	} else {
		fmt.Println(ui.TitleStyle.Render("✨ Welcome to Vaulty!"))
		fmt.Println(ui.MutedStyle.Render("  Secure secret management powered by GitHub"))
	}
	fmt.Println()

	cfg, err := config.Load("")
	if err != nil {
		cfg = &config.Config{}
	}

	if localMode {
		return runInitLocal(cfg)
	}

	return runInitGitHub(cfg)
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().BoolVarP(&localMode, "local", "l", false, "Use local storage instead of GitHub")
}
