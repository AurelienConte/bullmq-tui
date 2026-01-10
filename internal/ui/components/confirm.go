package components

import (
	"github.com/AurelienConte/bullmq-tui/internal/redis"
	"github.com/charmbracelet/lipgloss"
)

// ConfirmAction represents the type of action to execute
type ConfirmAction int

const (
	ConfirmActionNone ConfirmAction = iota
	ConfirmActionRetryJob
	ConfirmActionRetryAllFailed
	ConfirmActionDeleteJob
	ConfirmActionDrainQueue
	ConfirmActionPauseQueue
	ConfirmActionResumeQueue
)

type ConfirmDialog struct {
	title     string
	message   string
	selected  int // 0 = Yes, 1 = No
	width     int
	height    int
	action    ConfirmAction
	queueName string
	jobID     string
	jobState  redis.JobState
}

func NewConfirmDialog(title, message string) *ConfirmDialog {
	return &ConfirmDialog{
		title:    title,
		message:  message,
		selected: 1, // Default to No
		action:   ConfirmActionNone,
	}
}

// NewConfirmDialogWithAction creates a confirm dialog with action context
func NewConfirmDialogWithAction(title, message string, action ConfirmAction, queueName, jobID string, jobState redis.JobState) *ConfirmDialog {
	return &ConfirmDialog{
		title:     title,
		message:   message,
		selected:  1, // Default to No
		action:    action,
		queueName: queueName,
		jobID:     jobID,
		jobState:  jobState,
	}
}

func (c *ConfirmDialog) SetSize(width, height int) {
	c.width = width
	c.height = height
}

func (c *ConfirmDialog) ToggleSelection() {
	c.selected = (c.selected + 1) % 2
}

func (c *ConfirmDialog) IsYesSelected() bool {
	return c.selected == 0
}

// GetAction returns the action type
func (c *ConfirmDialog) GetAction() ConfirmAction {
	return c.action
}

// GetQueueName returns the queue name
func (c *ConfirmDialog) GetQueueName() string {
	return c.queueName
}

// GetJobID returns the job ID
func (c *ConfirmDialog) GetJobID() string {
	return c.jobID
}

// GetJobState returns the job state
func (c *ConfirmDialog) GetJobState() redis.JobState {
	return c.jobState
}

func (c ConfirmDialog) View() string {
	title := ModalTitleStyle.Render(c.title)

	message := StatLabelStyle.Render(c.message)

	// Buttons
	yesStyle := TabStyle.Copy().Padding(0, 3)
	noStyle := TabStyle.Copy().Padding(0, 3)

	if c.selected == 0 {
		yesStyle = TabActiveStyle.Copy().Padding(0, 3)
	} else {
		noStyle = TabActiveStyle.Copy().Padding(0, 3)
	}

	buttons := lipgloss.JoinHorizontal(
		lipgloss.Top,
		yesStyle.Render("Yes"),
		"  ",
		noStyle.Render("No"),
	)

	hint := StatLabelStyle.Render("←→ to select  •  Enter to confirm  •  Esc to cancel")

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		message,
		"",
		buttons,
		"",
		hint,
	)

	return ModalStyle.
		Width(50).
		Render(content)
}
