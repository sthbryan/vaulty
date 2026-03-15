package ui

import "github.com/charmbracelet/lipgloss"

const (
	Primary   = "#6366F1"
	Secondary = "#8B5CF6"
	Success   = "#10B981"
	Warning   = "#F59E0B"
	Error     = "#EF4444"
	Info      = "#3B82F6"
	Muted     = "#6B7280"
	Dark      = "#1F2937"
	Light     = "#F3F4F6"

	EnvColor      = "#10B981"
	SSHColor      = "#3B82F6"
	ResourceColor = "#F59E0B"
	ConfigColor   = "#EC4899"
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
				Foreground(lipgloss.Color("#9CA3AF"))

	TableRowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#D1D5DB"))

	TableRowAltStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#E5E7EB"))

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
