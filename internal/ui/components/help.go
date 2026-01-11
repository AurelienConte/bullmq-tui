package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type HelpOverlay struct {
	width  int
	height int
}

func NewHelpOverlay() *HelpOverlay {
	return &HelpOverlay{}
}

func (h *HelpOverlay) SetSize(width, height int) {
	h.width = width
	h.height = height
}

func (h HelpOverlay) View() string {
	title := ModalTitleStyle.Render("Keyboard Shortcuts")

	helpText := []string{
		"",
		KeyStyle.Render("Navigation"),
		"  ↑↓ / j k       Navigate up/down",
		"  ←→ / h l       Navigate left/right",
		"  tab / shift+tab  Switch between panels",
		"  1-5              Switch job state tabs",
		"",
		KeyStyle.Render("Actions"),
		"  a                Add job to selected queue",
		"  enter            View job details",
		"  r                Retry selected job",
		"  R                Retry all failed jobs",
		"  d                Delete selected job",
		"  D                Drain state / Clean all jobs",
		"  p                Pause/resume queue",
		"  ctrl+r           Force refresh",
		"",
		KeyStyle.Render("Other"),
		"  ?                Toggle this help",
		"  q / ctrl+c       Quit application",
		"",
		StatLabelStyle.Render("Press ? or Esc to close"),
	}

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		strings.Join(helpText, "\n"),
	)

	return ModalStyle.
		Width(50).
		Render(content)
}
