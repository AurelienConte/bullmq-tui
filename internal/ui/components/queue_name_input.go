package components

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type QueueNameInput struct {
	textInput textinput.Model
	width     int
	height    int
	err       string
}

func NewQueueNameInput() *QueueNameInput {
	ti := textinput.New()
	ti.Placeholder = "my-queue"
	ti.Focus()
	ti.CharLimit = 64
	ti.Width = 40

	return &QueueNameInput{
		textInput: ti,
	}
}

func (q *QueueNameInput) SetSize(width, height int) {
	q.width = width
	q.height = height
}

func (q *QueueNameInput) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	q.textInput, cmd = q.textInput.Update(msg)
	return cmd
}

func (q *QueueNameInput) Value() string {
	return q.textInput.Value()
}

func (q *QueueNameInput) SetError(err string) {
	q.err = err
}

func (q QueueNameInput) View() string {
	title := ModalTitleStyle.Render("Create Queue")
	label := StatLabelStyle.Render("Queue Name:")

	errorMsg := ""
	if q.err != "" {
		errorStyle := StatLabelStyle.Copy().Foreground(lipgloss.Color("#EF4444"))
		errorMsg = errorStyle.Render("⚠ " + q.err)
	}

	hint := StatLabelStyle.Render("Enter to submit  •  Esc to cancel")

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		label,
		q.textInput.View(),
		errorMsg,
		"",
		hint,
	)

	return ModalStyle.Width(50).Render(content)
}
