package ui

import (
	"fmt"

	"github.com/charmbracelet/huh"
)

// AskPassword prompts for a password with hidden input
func AskPassword(title string) (string, error) {
	var password string

	err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(title).
				EchoMode(huh.EchoModePassword).
				Placeholder("Enter your password...").
				Value(&password).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("password cannot be empty")
					}
					return nil
				}),
		),
	).Run()

	if err != nil {
		return "", err
	}

	return password, nil
}

// AskChoice prompts the user to select from a list of options
func AskChoice(title string, options []string) (string, error) {
	if len(options) == 0 {
		return "", fmt.Errorf("no options provided")
	}

	var selected string

	// Convert strings to huh.Option
	huhOptions := make([]huh.Option[string], len(options))
	for i, opt := range options {
		huhOptions[i] = huh.NewOption(opt, opt)
	}

	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title(title).
				Options(huhOptions...).
				Value(&selected),
		),
	).Run()

	if err != nil {
		return "", err
	}

	return selected, nil
}

// AskInput prompts for text input with an optional placeholder
func AskInput(title string, placeholder string) (string, error) {
	var input string

	err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(title).
				Placeholder(placeholder).
				Value(&input).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("input cannot be empty")
					}
					return nil
				}),
		),
	).Run()

	if err != nil {
		return "", err
	}

	return input, nil
}

// AskInputOptional prompts for text input (can be empty)
func AskInputOptional(title string, placeholder string) (string, error) {
	var input string

	err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(title).
				Placeholder(placeholder).
				Value(&input),
		),
	).Run()

	if err != nil {
		return "", err
	}

	return input, nil
}

// AskConfirm prompts for a yes/no confirmation
func AskConfirm(title string, defaultYes bool) (bool, error) {
	var confirmed bool

	err := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(title).
				Value(&confirmed).
				Affirmative("Yes").
				Negative("No"),
		),
	).Run()

	if err != nil {
		return defaultYes, err
	}

	return confirmed, nil
}

// AskConfirmWithDefault prompts for confirmation with a default value
func AskConfirmWithDefault(title string, defaultValue bool) (bool, error) {
	confirmed := defaultValue

	err := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(title).
				Value(&confirmed).
				Affirmative("Yes").
				Negative("No"),
		),
	).Run()

	if err != nil {
		return defaultValue, err
	}

	return confirmed, nil
}
