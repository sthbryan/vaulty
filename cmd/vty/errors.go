package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/sthbryan/vaulty/v2/internal/ui"
	"github.com/sthbryan/vaulty/v2/internal/vault"
)

type CommandError struct {
	Message  string
	Hint     string
	Examples []string
}

func (e *CommandError) Error() string {
	return e.Message
}

func handleCommandError(err error) {
	if err == nil {
		return
	}

	errStr := err.Error()

	if strings.HasPrefix(errStr, "unknown command") {
		return
	}

	var vaultErr *vault.VaultError
	if errors.As(err, &vaultErr) {
		ui.PrintError(vaultErr.Message)
		if vaultErr.Hint != "" {
			ui.PrintInfo("")
			ui.PrintInfo(vaultErr.Hint)
		}
		return
	}

	var cmdErr *CommandError
	if errors.As(err, &cmdErr) {
		ui.PrintError(cmdErr.Message)
		if cmdErr.Hint != "" {
			ui.PrintInfo("")
			ui.PrintInfo(cmdErr.Hint)
		}
		if len(cmdErr.Examples) > 0 {
			ui.PrintInfo("")
			ui.PrintBold("Examples:")
			for _, ex := range cmdErr.Examples {
				ui.PrintInfo(ex)
			}
		}
		return
	}

	ui.PrintError(fmt.Sprintf("Error: %v", err))
}
