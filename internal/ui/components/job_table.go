package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/AurelienConte/bullmq-tui/internal/redis"
	"github.com/charmbracelet/lipgloss"
)

type JobTable struct {
	jobs              []redis.Job
	cursor            int
	focused           bool
	width             int
	height            int
	offset            int
	selectedJobID     string // Track selected job ID for preservation across refreshes
}

func NewJobTable() JobTable {
	return JobTable{
		cursor: 0,
		offset: 0,
	}
}

func (jt *JobTable) SetJobs(jobs []redis.Job) {
	// Store currently selected job ID before updating
	previousJobID := jt.selectedJobID

	jt.jobs = jobs

	// Try to restore selection by finding the previously selected job
	if previousJobID != "" && len(jobs) > 0 {
		for i, job := range jobs {
			if job.ID == previousJobID {
				jt.cursor = i
				// Adjust offset to keep selected job visible
				jt.adjustOffset()
				return
			}
		}
	}

	// If job wasn't found or no previous selection, keep cursor in bounds
	if jt.cursor >= len(jobs) {
		jt.cursor = len(jobs) - 1
	}
	if jt.cursor < 0 {
		jt.cursor = 0
	}

	// Update selectedJobID to current cursor position
	if jt.cursor >= 0 && jt.cursor < len(jobs) {
		jt.selectedJobID = jobs[jt.cursor].ID
	} else {
		jt.selectedJobID = ""
	}

	jt.adjustOffset()
}

// adjustOffset ensures the cursor is visible in the viewport
func (jt *JobTable) adjustOffset() {
	visibleRows := jt.height - 3 // Account for header
	if visibleRows < 1 {
		visibleRows = 1
	}

	// If cursor is above viewport, scroll up
	if jt.cursor < jt.offset {
		jt.offset = jt.cursor
	}

	// If cursor is below viewport, scroll down
	if jt.cursor >= jt.offset+visibleRows {
		jt.offset = jt.cursor - visibleRows + 1
	}

	// Ensure offset is not negative
	if jt.offset < 0 {
		jt.offset = 0
	}
}

func (jt *JobTable) SetFocused(focused bool) {
	jt.focused = focused
}

func (jt *JobTable) SetSize(width, height int) {
	jt.width = width
	jt.height = height
}

func (jt *JobTable) SelectedJob() *redis.Job {
	if jt.cursor < len(jt.jobs) && jt.cursor >= 0 {
		return &jt.jobs[jt.cursor]
	}
	return nil
}

func (jt *JobTable) MoveUp() {
	if jt.cursor > 0 {
		jt.cursor--
		// Update selected job ID
		if jt.cursor >= 0 && jt.cursor < len(jt.jobs) {
			jt.selectedJobID = jt.jobs[jt.cursor].ID
		}
		// Adjust offset if needed
		if jt.cursor < jt.offset {
			jt.offset = jt.cursor
		}
	}
}

func (jt *JobTable) MoveDown() {
	if jt.cursor < len(jt.jobs)-1 {
		jt.cursor++
		// Update selected job ID
		if jt.cursor >= 0 && jt.cursor < len(jt.jobs) {
			jt.selectedJobID = jt.jobs[jt.cursor].ID
		}
		// Adjust offset if needed
		visibleRows := jt.height - 3 // Account for header
		if jt.cursor >= jt.offset+visibleRows {
			jt.offset = jt.cursor - visibleRows + 1
		}
	}
}

func (jt JobTable) View() string {
	// Table header
	header := renderTableRow(
		[]string{"ID", "Name", "Attempts", "Created", "Status"},
		[]int{10, 30, 10, 20, 15},
		TableHeaderStyle,
		false,
	)

	var rows []string
	rows = append(rows, header)

	if len(jt.jobs) == 0 {
		emptyMsg := StatLabelStyle.Render("No jobs in this state")
		rows = append(rows, "", emptyMsg)
	} else {
		// Calculate visible range
		visibleRows := jt.height - 3
		if visibleRows < 1 {
			visibleRows = 1
		}

		start := jt.offset
		end := start + visibleRows
		if end > len(jt.jobs) {
			end = len(jt.jobs)
		}

		for i := start; i < end; i++ {
			job := jt.jobs[i]
			isSelected := i == jt.cursor && jt.focused

			// Format job data
			id := truncate(job.ID, 10)
			name := truncate(job.Name, 30)
			attempts := fmt.Sprintf("%d/%d", job.AttemptsMade, job.Attempts)
			created := formatTime(job.Timestamp)
			status := formatJobStatus(job)

			row := renderTableRow(
				[]string{id, name, attempts, created, status},
				[]int{10, 30, 10, 20, 15},
				TableRowStyle,
				isSelected,
			)

			rows = append(rows, row)
		}
	}

	content := lipgloss.JoinVertical(lipgloss.Left, rows...)

	// Choose border color and style based on focus state
	borderColor := unfocusedBorderColor
	borderStyle := lipgloss.RoundedBorder()
	if jt.focused {
		borderColor = focusedBorderColor
		borderStyle = lipgloss.ThickBorder()
	}

	return lipgloss.NewStyle().
		Width(jt.width).
		Height(jt.height).
		Border(borderStyle).
		BorderForeground(borderColor).
		Padding(0, 1).
		Render(content)
}

func renderTableRow(cells []string, widths []int, style lipgloss.Style, selected bool) string {
	var parts []string
	for i, cell := range cells {
		width := widths[i]
		if i < len(widths) {
			cellStyle := lipgloss.NewStyle().Width(width)
			if selected {
				cellStyle = TableRowSelectedStyle.Width(width)
			}
			parts = append(parts, cellStyle.Render(truncate(cell, width)))
		}
	}

	row := strings.Join(parts, " ")
	if selected {
		return TableRowSelectedStyle.Render(row)
	}
	return style.Render(row)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func formatTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	if diff < time.Minute {
		return "just now"
	}
	if diff < time.Hour {
		return fmt.Sprintf("%dm ago", int(diff.Minutes()))
	}
	if diff < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(diff.Hours()))
	}
	return t.Format("Jan 02 15:04")
}

func formatJobStatus(job redis.Job) string {
	switch job.State {
	case redis.JobStateWaiting:
		return StatLabelStyle.Render("Waiting")
	case redis.JobStateActive:
		return StatValueStyle.
			Foreground(lipgloss.Color("#10B981")).
			Render("Active")
	case redis.JobStateDelayed:
		return StatLabelStyle.Render("Delayed")
	case redis.JobStateCompleted:
		return StatValueStyle.
			Foreground(lipgloss.Color("#10B981")).
			Render("Completed")
	case redis.JobStateFailed:
		return StatValueStyle.
			Foreground(lipgloss.Color("#EF4444")).
			Render("Failed")
	default:
		return string(job.State)
	}
}
