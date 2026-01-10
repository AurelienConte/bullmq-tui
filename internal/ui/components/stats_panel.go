package components

import (
	"fmt"

	"github.com/AurelienConte/bullmq-tui/internal/redis"
	"github.com/AurelienConte/bullmq-tui/internal/stats"
	"github.com/charmbracelet/lipgloss"
)

type StatsPanel struct {
	stats     *stats.QueueStats
	counts    redis.QueueCounts
	sparkline Sparkline
	tabs      Tabs
	width     int
	height    int
	focused   bool
}

func NewStatsPanel() StatsPanel {
	return StatsPanel{
		sparkline: NewSparkline(40),
		tabs:      NewTabs(),
		focused:   false,
	}
}

func (s *StatsPanel) SetFocused(focused bool) {
	s.focused = focused
}

func (s *StatsPanel) SetStats(st *stats.QueueStats) {
	s.stats = st
	if st != nil {
		s.sparkline.SetValues(st.ThroughputHistory)
	}
}

func (s *StatsPanel) SetCounts(counts redis.QueueCounts) {
	s.counts = counts
	s.tabs.SetCounts(counts)
}

func (s *StatsPanel) SetActiveState(state redis.JobState) {
	s.tabs.SetActive(state)
}

func (s *StatsPanel) GetActiveState() redis.JobState {
	return s.tabs.GetActiveState()
}

func (s *StatsPanel) NextTab() {
	s.tabs.Next()
}

func (s *StatsPanel) PreviousTab() {
	s.tabs.Previous()
}

func (s *StatsPanel) SetSize(width, height int) {
	s.width = width
	s.height = height
	s.sparkline.SetWidth(width - 30)
	s.tabs.SetWidth(width)
}

func (s StatsPanel) View() string {
	// Tabs
	tabsView := s.tabs.View()

	// Stats row
	var statsItems []string

	// Queue counts summary
	total := s.counts.Waiting + s.counts.Active + s.counts.Delayed +
		s.counts.Completed + s.counts.Failed

	statsItems = append(statsItems,
		StatLabelStyle.Render("Total: ")+
			StatValueStyle.Render(fmt.Sprintf("%d", total)))

	// Throughput
	if s.stats != nil {
		statsItems = append(statsItems,
			StatLabelStyle.Render("Jobs/min: ")+
				StatValueStyle.Render(fmt.Sprintf("%.1f", s.stats.JobsPerMinute)))

		// Failure rate
		if s.stats.FailureRate > 0 {
			failStyle := StatValueStyle.Copy().
				Foreground(lipgloss.Color("#EF4444"))
			statsItems = append(statsItems,
				StatLabelStyle.Render("Failure: ")+
					failStyle.Render(fmt.Sprintf("%.1f%%", s.stats.FailureRate)))
		}
	}

	statsRow := lipgloss.JoinHorizontal(lipgloss.Top,
		statsItems[0],
		"  │  ",
	)
	for i := 1; i < len(statsItems); i++ {
		statsRow = lipgloss.JoinHorizontal(lipgloss.Top,
			statsRow,
			statsItems[i],
			"  │  ",
		)
	}

	// Sparkline
	sparklineView := ""
	if s.stats != nil && len(s.stats.ThroughputHistory) > 0 {
		sparklineView = StatLabelStyle.Render("Throughput: ") +
			s.sparkline.View()
	}

	// Combine all
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		tabsView,
		"",
		statsRow,
		sparklineView,
	)

	// Choose border color and style based on focus state
	borderColor := unfocusedBorderColor
	borderStyle := lipgloss.RoundedBorder()
	if s.focused {
		borderColor = focusedBorderColor
		borderStyle = lipgloss.ThickBorder()
	}

	return StatsPanelStyle.Copy().
		Border(borderStyle).
		BorderForeground(borderColor).
		Width(s.width).
		Render(content)
}
