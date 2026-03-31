package ui

import (
	"fmt"

	"github.com/charmbracelet/huh"
)

func ConfirmOrAbort(message string) error {
	confirmed, err := AskConfirm(message, false)
	if err != nil {
		return fmt.Errorf("confirmation failed: %w", err)
	}
	if !confirmed {
		PrintInfo("Operation cancelled")
		return fmt.Errorf("operation aborted by user")
	}
	return nil
}

func ConfirmOrAbortWithDefault(message string, defaultValue bool) error {
	confirmed, err := AskConfirmWithDefault(message, defaultValue)
	if err != nil {
		return fmt.Errorf("confirmation failed: %w", err)
	}
	if !confirmed {
		PrintInfo("Operation cancelled")
		return fmt.Errorf("operation aborted by user")
	}
	return nil
}

func AskConfirmInteractive(message string) (bool, error) {
	var confirmed bool
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(message).
				Affirmative("Yes").
				Negative("No"),
		),
	).Run()
	return confirmed, err
}
