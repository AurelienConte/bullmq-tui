package components

import (
	"fmt"

	"github.com/AurelienConte/bullmq-tui/internal/redis"
	"github.com/charmbracelet/lipgloss"
)

type Sidebar struct {
	queues  []redis.Queue
	cursor  int
	focused bool
	width   int
	height  int
}

func NewSidebar() Sidebar {
	return Sidebar{
		cursor: 0,
	}
}

func (s *Sidebar) SetQueues(queues []redis.Queue) {
	s.queues = queues
	// Keep cursor in bounds
	if s.cursor >= len(queues) {
		s.cursor = len(queues) - 1
	}
	if s.cursor < 0 {
		s.cursor = 0
	}
}

func (s *Sidebar) SetFocused(focused bool) {
	s.focused = focused
}

func (s *Sidebar) SetSize(width, height int) {
	s.width = width
	s.height = height
}

func (s *Sidebar) SelectedQueue() string {
	if s.cursor < len(s.queues) && s.cursor >= 0 {
		return s.queues[s.cursor].Name
	}
	return ""
}

func (s *Sidebar) MoveUp() {
	if s.cursor > 0 {
		s.cursor--
	}
}

func (s *Sidebar) MoveDown() {
	if s.cursor < len(s.queues)-1 {
		s.cursor++
	}
}

func (s Sidebar) View() string {
	var items []string

	title := HeaderStyle.Render("QUEUES")
	items = append(items, title, "")

	if len(s.queues) == 0 {
		items = append(items, StatLabelStyle.Render("No queues found"))
	}

	for i, queue := range s.queues {
		// Format: "> queue-name    [!]"
		name := queue.Name
		indicator := "   "
		if queue.Counts.Failed > 0 {
			indicator = StatValueStyle.Foreground(lipgloss.Color("#EF4444")).Render("[!]")
		}

		pausedStr := ""
		if queue.IsPaused {
			pausedStr = StatLabelStyle.Render(" [PAUSED]")
		}

		line := fmt.Sprintf("%s%s %s", name, pausedStr, indicator)

		// Apply style based on state
		var style lipgloss.Style
		if i == s.cursor && s.focused {
			style = QueueItemSelectedStyle
		} else if queue.Counts.Failed > 0 {
			style = QueueItemWithFailuresStyle
		} else {
			style = QueueItemStyle
		}

		// Add cursor indicator
		cursor := "  "
		if i == s.cursor {
			cursor = "> "
		}

		items = append(items, style.Render(cursor+line))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, items...)

	// Choose border color and style based on focus state
	borderColor := unfocusedBorderColor
	borderStyle := lipgloss.RoundedBorder()
	if s.focused {
		borderColor = focusedBorderColor
		borderStyle = lipgloss.ThickBorder()
	}

	return SidebarStyle.
		Border(borderStyle).
		BorderForeground(borderColor).
		Width(s.width).
		Height(s.height).
		Render(content)
}
