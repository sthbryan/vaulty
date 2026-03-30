package ui

import "github.com/charmbracelet/lipgloss"

const (
	Primary   = "#F97316"
	Secondary = "#DC2626"
	Success   = "#22C55E"
	Warning   = "#F59E0B"
	Error     = "#EF4444"
	Info      = "#EA580C"
	Muted     = "#9CA3AF"
	Dark      = "#1F1F1F"
	Light     = "#FEF3E2"

	EnvColor      = "#F97316"
	SSHColor      = "#DC2626"
	ResourceColor = "#F59E0B"
	ConfigColor   = "#B45309"
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

	SectionTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color(Primary)).
				MarginBottom(1)

	TableHeaderNewStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color(Warning))

	TableRowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(Light))

	TableRowAltStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(ResourceColor))

	TypeEnvStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(EnvColor)).
			Bold(true)

	TypeSSHStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(SSHColor)).
			Bold(true)

	TypeResourceStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(ResourceColor)).
				Bold(true)

	TypeConfigStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ConfigColor)).
			Bold(true)
)
