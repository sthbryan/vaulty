package ui

import "github.com/charmbracelet/lipgloss"

const (
	Primary = "#7C3AED"
	Success = "#10B981"
	Warning = "#F59E0B"
	Error   = "#EF4444"
	Info    = "#3B82F6"
	Muted   = "#6B7280"
)

var (
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(Primary)).
			MarginBottom(1)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(Success))

	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(Error))

	WarningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(Warning))

	InfoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(Info))

	MutedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(Muted))

	TableStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(Muted))

	TableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color(Primary))

	BoxStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			Padding(1, 2)

	HighlightStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(Primary))
)
