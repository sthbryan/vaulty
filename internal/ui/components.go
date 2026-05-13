package ui

import (
	"fmt"
	"os/exec"
	"strings"

	"charm.land/huh/v2"
)

type SelectOption struct {
	ID    string
	Label string
}

func Select(title string, options []SelectOption) (string, error) {
	var value string

	opts := make([]huh.Option[string], len(options))
	for i, opt := range options {
		opts[i] = huh.NewOption(opt.Label, opt.ID)
	}

	step := huh.NewSelect[string]().
		Title(title).
		Options(opts...).
		Value(&value).
		WithTheme(Theme)

	if err := step.Run(); err != nil {
		return "", err
	}
	return value, nil
}

func Input(title, placeholder string) (string, error) {
	var value string
	step := huh.NewInput().
		Title(title).
		Placeholder(placeholder).
		Value(&value).
		WithTheme(Theme)

	if err := step.Run(); err != nil {
		return "", err
	}
	return value, nil
}

func Password(title, placeholder string) (string, error) {
	var value string
	step := huh.NewInput().
		Title(title).
		Placeholder(placeholder).
		EchoMode(huh.EchoModePassword).
		Value(&value).
		WithTheme(Theme)

	if err := step.Run(); err != nil {
		return "", err
	}
	return value, nil
}

func Confirm(title string) (bool, error) {
	var value bool
	selector := huh.NewSelect[bool]().
		Title(title).
		Options(
			huh.NewOption("Yes", true),
			huh.NewOption("No", false),
		).
		Value(&value).
		WithTheme(Theme)

	if err := selector.Run(); err != nil {
		return false, err
	}
	return value, nil
}

// --- Group builders ---

func SelectGroup(title, key string, options []SelectOption, value *string) *huh.Group {
	opts := make([]huh.Option[string], len(options))
	for i, opt := range options {
		opts[i] = huh.NewOption(opt.Label, opt.ID)
	}
	return huh.NewGroup(
		huh.NewSelect[string]().
			Title(title).
			Key(key).
			Options(opts...).
			Value(value),
	)
}

func InputGroup(title, key, placeholder string, value *string) *huh.Group {
	return huh.NewGroup(
		huh.NewInput().
			Title(title).
			Key(key).
			Placeholder(placeholder).
			Value(value),
	)
}

func PasswordGroup(title, key, placeholder string, value *string) *huh.Group {
	return huh.NewGroup(
		huh.NewInput().
			Title(title).
			Key(key).
			Placeholder(placeholder).
			EchoMode(huh.EchoModePassword).
			Value(value),
	)
}

func ConfirmGroup(title, key string, value *bool) *huh.Group {
	return huh.NewGroup(
		huh.NewSelect[bool]().
			Title(title).
			Key(key).
			Options(
				huh.NewOption("Yes", true),
				huh.NewOption("No", false),
			).
			Value(value),
	)
}

// --- Print functions ---

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
	fmt.Println(BoldStyle.Render(msg))
}

func ValidatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}
	return nil
}

func GetGitHubTokenCLI() (string, error) {
	cmd := exec.Command("gh", "auth", "token")
	output, err := cmd.Output()
	if err != nil || len(output) == 0 {
		return "", fmt.Errorf("GitHub CLI not available")
	}
	return strings.TrimSpace(string(output)), nil
}
