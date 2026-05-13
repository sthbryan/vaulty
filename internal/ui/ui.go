package ui

import (
	"fmt"

	"charm.land/huh/v2"
)

type SelectOption struct {
	ID    string
	Label string
}

func PromptPassword(title, placeholder string) (string, error) {
	var value string
	input := huh.NewInput().
		Title(title).
		Placeholder(placeholder).
		Prompt("▶ ").
		EchoMode(huh.EchoModePassword).
		Value(&value).
		WithTheme(theme())

	if err := input.Run(); err != nil {
		return "", err
	}
	return value, nil
}

func PromptInput(title, placeholder string) (string, error) {
	var value string
	input := huh.NewInput().
		Title(title).
		Placeholder(placeholder).
		Prompt("▶ ").
		Value(&value).
		WithTheme(theme())

	if err := input.Run(); err != nil {
		return "", err
	}
	return value, nil
}

func PromptSelect(title string, options []SelectOption) (string, error) {
	var value string

	opts := make([]huh.Option[string], len(options))
	for i, opt := range options {
		opts[i] = huh.NewOption(opt.Label, opt.ID)
	}

	step := huh.NewSelect[string]().
		Title(title).
		Options(opts...).
		Value(&value).
		WithTheme(theme())

	if err := step.Run(); err != nil {
		return "", err
	}
	return value, nil
}

func PromptConfirm(title string) (bool, error) {
	var value bool
	confirm := huh.NewConfirm().
		Title(title).
		Value(&value).
		WithTheme(theme())

	if err := confirm.Run(); err != nil {
		return false, err
	}
	return value, nil
}

func ValidatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}
	return nil
}

func PrintSuccess(msg string) {
	fmt.Println(SuccessStyle.Render("[OK] " + msg))
}

func PrintError(msg string) {
	fmt.Println(ErrStyle.Render("[!!] " + msg))
}

func PrintInfo(msg string) {
	fmt.Println(InfoStyle.Render("[..] " + msg))
}

func PrintBold(msg string) {
	fmt.Println(BoldStyle.Render("[>>] " + msg))
}
