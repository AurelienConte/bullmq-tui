package components

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	primaryColor   = lipgloss.Color("#7C3AED")   // Purple
	secondaryColor = lipgloss.Color("#10B981")   // Green
	dangerColor    = lipgloss.Color("#EF4444")   // Red
	warningColor   = lipgloss.Color("#F59E0B")   // Yellow
	mutedColor     = lipgloss.Color("#6B7280")   // Gray
	bgColor        = lipgloss.Color("#1F2937")   // Dark gray
	textColor      = lipgloss.Color("#F3F4F6")   // Light gray

	// Base styles
	BaseStyle = lipgloss.NewStyle()

	// Header
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			Padding(0, 1)

	// Sidebar
	SidebarStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(mutedColor).
			Padding(1).
			Width(30)

	QueueItemStyle = lipgloss.NewStyle().
			Padding(0, 1)

	QueueItemSelectedStyle = QueueItemStyle.Copy().
				Background(primaryColor).
				Foreground(lipgloss.Color("#FFFFFF"))

	QueueItemWithFailuresStyle = QueueItemStyle.Copy().
					Foreground(dangerColor)

	// Stats panel
	StatsPanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(mutedColor).
			Padding(1)

	StatLabelStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	StatValueStyle = lipgloss.NewStyle().
			Bold(true)

	// Job table
	TableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(mutedColor).
				Border(lipgloss.NormalBorder(), false, false, true, false).
				BorderForeground(mutedColor)

	TableRowStyle = lipgloss.NewStyle().
			Padding(0, 1)

	TableRowSelectedStyle = TableRowStyle.Copy().
				Background(primaryColor).
				Foreground(lipgloss.Color("#FFFFFF"))

	// Status bar
	StatusBarStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Padding(0, 1)

	KeyStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true)

	// Modal
	ModalStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(1, 2).
			Width(60)

	ModalTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			MarginBottom(1)

	// Toast
	ToastStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Background(secondaryColor).
			Foreground(lipgloss.Color("#FFFFFF"))

	ToastErrorStyle = ToastStyle.Copy().
			Background(dangerColor)

	// Tabs
	TabStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Foreground(mutedColor)

	TabActiveStyle = TabStyle.Copy().
			Background(primaryColor).
			Foreground(lipgloss.Color("#FFFFFF"))
)
