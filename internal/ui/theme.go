package ui

import (
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
)

var (
	BoldStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#ff0055"))
	SuccessStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff9f"))
	ErrStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0055"))
	InfoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#00d4ff"))
	WarningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffcc00"))
	MutedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))
)

func theme() huh.ThemeFunc {
	return huh.ThemeFunc(func(isDark bool) *huh.Styles {
		styles := huh.ThemeBase(isDark)

		styles.Blurred.Base = lipgloss.NewStyle().Foreground(lipgloss.Color("#2b2a33"))
		styles.Focused.Base = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff2d95"))

		styles.Blurred.Title = lipgloss.NewStyle().Foreground(lipgloss.Color("#8a2b2b"))
		styles.Focused.Title = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff3b3b"))

		styles.Focused.SelectSelector = lipgloss.NewStyle().SetString("▶ ").Foreground(lipgloss.Color("#ff7bb0"))
		styles.Blurred.SelectSelector = lipgloss.NewStyle().SetString("  ").Foreground(lipgloss.Color("#d5d3d8"))

		styles.Focused.MultiSelectSelector = lipgloss.NewStyle().SetString("▶ ").Foreground(lipgloss.Color("#ff7bb0"))
		styles.Blurred.MultiSelectSelector = lipgloss.NewStyle().SetString("  ").Foreground(lipgloss.Color("#d5d3d8"))

		styles.Focused.SelectedOption = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff7bb0"))
		styles.Focused.UnselectedOption = lipgloss.NewStyle().Foreground(lipgloss.Color("#6b6b73"))
		styles.Blurred.SelectedOption = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff7bb0"))
		styles.Blurred.UnselectedOption = lipgloss.NewStyle().Foreground(lipgloss.Color("#6b6b73"))

		styles.Focused.TextInput.Text = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff3b3b"))
		styles.Focused.TextInput.Placeholder = lipgloss.NewStyle().Foreground(lipgloss.Color("#6b6b73"))

		styles.Focused.FocusedButton = lipgloss.NewStyle().Foreground(lipgloss.Color("#00f2ff"))
		styles.Blurred.FocusedButton = lipgloss.NewStyle().Foreground(lipgloss.Color("#8a2be2"))

		return styles
	})
}
