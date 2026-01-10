package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type StatusBar struct {
	width   int
	message string
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

func (s StatusBar) View() string {
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

	keysStr := strings.Join(keys, " │ ")

	// Show custom message if set
	if s.message != "" {
		keysStr = s.message
	}

	return StatusBarStyle.
		Width(s.width).
		Border(lipgloss.NormalBorder(), true, false, false, false).
		BorderForeground(HeaderStyle.GetForeground()).
		Render(keysStr)
}
