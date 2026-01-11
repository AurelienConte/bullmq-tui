package components

import (
	"fmt"
	"strings"

	"github.com/AurelienConte/bullmq-tui/internal/redis"
	"github.com/charmbracelet/lipgloss"
)

type Tabs struct {
	tabs        []TabItem
	activeIndex int
	width       int
}

type TabItem struct {
	Label string
	State redis.JobState
	Count int64
}

func NewTabs() Tabs {
	return Tabs{
		tabs: []TabItem{
			{Label: "Waiting", State: redis.JobStateWaiting},
			{Label: "Active", State: redis.JobStateActive},
			{Label: "Delayed", State: redis.JobStateDelayed},
			{Label: "Completed", State: redis.JobStateCompleted},
			{Label: "Failed", State: redis.JobStateFailed},
		},
		activeIndex: 0,
	}
}

func (t *Tabs) SetCounts(counts redis.QueueCounts) {
	countMap := map[redis.JobState]int64{
		redis.JobStateWaiting:   counts.Waiting,
		redis.JobStateActive:    counts.Active,
		redis.JobStateDelayed:   counts.Delayed,
		redis.JobStateCompleted: counts.Completed,
		redis.JobStateFailed:    counts.Failed,
	}

	for i := range t.tabs {
		t.tabs[i].Count = countMap[t.tabs[i].State]
	}
}

func (t *Tabs) SetActive(state redis.JobState) {
	for i, tab := range t.tabs {
		if tab.State == state {
			t.activeIndex = i
			break
		}
	}
}

func (t *Tabs) GetActiveState() redis.JobState {
	if t.activeIndex < len(t.tabs) {
		return t.tabs[t.activeIndex].State
	}
	return redis.JobStateWaiting
}

func (t *Tabs) Next() {
	t.activeIndex = (t.activeIndex + 1) % len(t.tabs)
}

func (t *Tabs) Previous() {
	t.activeIndex = (t.activeIndex - 1 + len(t.tabs)) % len(t.tabs)
}

func (t *Tabs) SetWidth(width int) {
	t.width = width
}

func (t Tabs) View() string {
	var renderedTabs []string

	for i, tab := range t.tabs {
		label := tab.Label
		if tab.Count > 0 {
			label = label + " (" + formatCount(tab.Count) + ")"
		}

		style := TabStyle
		if i == t.activeIndex {
			style = TabActiveStyle
		}

		renderedTabs = append(renderedTabs, style.Render(label))
	}

	return strings.Join(renderedTabs, " ")
}

func formatCount(n int64) string {
	if n >= 1000000 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F59E0B")).
			Render("999k+")
	}
	if n >= 1000 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F59E0B")).
			Render(fmt.Sprintf("%dk", n/1000))
	}
	return fmt.Sprintf("%d", n)
}
