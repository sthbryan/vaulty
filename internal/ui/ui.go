package ui

import (
	"fmt"
	"time"

	"charm.land/bubbles/v2/spinner"
	"charm.land/huh/v2"
)

type SelectOption struct {
	ID    string
	Label string
}

type spinnerState struct {
	s       spinner.Model
	done    chan struct{}
	running bool
}

func newSpinner() *spinnerState {
	s := spinner.New(spinner.WithSpinner(spinner.Line))
	return &spinnerState{s: s, done: make(chan struct{})}
}

func (sp *spinnerState) Start(msg string) {
	sp.running = true
	go func() {
		for {
			select {
			case <-sp.done:
				return
			case <-time.After(100 * time.Millisecond):
				sp.s, _ = sp.s.Update(spinner.TickMsg{})
				fmt.Printf("\r%s %s", sp.s.View(), InfoStyle.Render(msg))
			}
		}
	}()
}

func (sp *spinnerState) Stop() {
	if sp.running {
		sp.done <- struct{}{}
		sp.running = false
		fmt.Printf("\r                    \r")
	}
}

func PrintSpinner(msg string) func() {
	sp := newSpinner()
	sp.Start(msg)
	return sp.Stop
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

func PromptInput(title, placeholder, defaultValue string) (string, error) {
	var value string
	input := huh.NewInput().
		Title(title).
		Placeholder(placeholder).
		Prompt("▶ ").
		Value(&value)

	if defaultValue != "" {
		value = defaultValue
		input.Placeholder(defaultValue)
	}

	input.WithTheme(theme())

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
	fmt.Println(BoldStyle.Render(msg))
}
