package ui

import (
	"fmt"

	"github.com/charmbracelet/huh"
)

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

func AskChoice(title string, options []string) (string, error) {
	if len(options) == 0 {
		return "", fmt.Errorf("no options provided")
	}

	var selected string

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
