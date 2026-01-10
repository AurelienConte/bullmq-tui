package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type StatusBar struct {
	width       int
	message     string
	focusedArea string // "Queues" or "Jobs"
}

func NewStatusBar() StatusBar {
	return StatusBar{}
}

func (s *StatusBar) SetSize(width int) {
	s.width = width
}

func (s *StatusBar) SetMessage(msg string) {
	s.message = msg
}

func (s *StatusBar) SetFocusedArea(area string) {
	s.focusedArea = area
}

func (s StatusBar) View() string {
	// Focus indicator with high contrast colors
	focusIndicator := ""
	if s.focusedArea != "" {
		focusIndicator = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#F59E0B")). // Bright yellow/orange
			Render("Focus: ") +
			lipgloss.NewStyle().
			Bold(true).
			Background(lipgloss.Color("#7C3AED")). // Purple background
			Foreground(lipgloss.Color("#FFFFFF")). // White text
			Padding(0, 1).
			Render(s.focusedArea) + " │ "
	}

	// Default keybindings
	keys := []string{
		KeyStyle.Render("tab") + " switch",
		KeyStyle.Render("↑↓/jk") + " navigate",
		KeyStyle.Render("enter") + " view",
		KeyStyle.Render("r") + " retry",
		KeyStyle.Render("d") + " delete",
		KeyStyle.Render("p") + " pause",
		KeyStyle.Render("?") + " help",
		KeyStyle.Render("q") + " quit",
	}

	keysStr := focusIndicator + strings.Join(keys, " │ ")

	// Show custom message if set (but keep focus indicator)
	if s.message != "" {
		keysStr = focusIndicator + s.message
	}

	return StatusBarStyle.
		Width(s.width).
		Border(lipgloss.NormalBorder(), true, false, false, false).
		BorderForeground(HeaderStyle.GetForeground()).
		Render(keysStr)
}
