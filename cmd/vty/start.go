package main

import (
	"github.com/spf13/cobra"
	"github.com/sthbryan/vaulty/v2/internal/start"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start vaulty (init or unlock)",
	Long:  `Create a new vault or unlock an existing one.`,
	RunE:  runStart,
}

var startSessionDuration string

func init() {
	rootCmd.AddCommand(startCmd)
	startCmd.Flags().StringVar(&startSessionDuration, "session", "8h", "Session duration (8h, 24h, 7d, 30d)")
}

func runStart(cmd *cobra.Command, args []string) error {
	runner := start.NewRunner(startSessionDuration)
	return runner.Run()
}