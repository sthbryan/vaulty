package ui

import "github.com/charmbracelet/lipgloss"

// Color palette constants
const (
	Primary = "#7C3AED" // Purple
	Success = "#10B981" // Green
	Warning = "#F59E0B" // Amber
	Error   = "#EF4444" // Red
	Info    = "#3B82F6" // Blue
	Muted   = "#6B7280" // Gray
)

// Lipgloss styles
var (
	// Title style for headers and important text
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(Primary)).
			MarginBottom(1)

	// Success style for positive feedback
	SuccessStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(Success))

	// Error style for errors and failures
	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(Error))

	// Warning style for warnings and cautions
	WarningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(Warning))

	// Info style for informational messages
	InfoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(Info))

	// Muted style for secondary/de-emphasized text
	MutedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(Muted))

	// Table style for data tables
	TableStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(Muted))

	// TableHeader style for table headers
	TableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color(Primary))

	// Box style for bordered containers
	BoxStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			Padding(1, 2)
)
