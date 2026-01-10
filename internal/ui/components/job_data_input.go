package components

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type JobDataInput struct {
	textarea  textarea.Model
	queueName string
	width     int
	height    int
	err       string
}

func NewJobDataInput(queueName string) *JobDataInput {
	ta := textarea.New()
	ta.Placeholder = `{
  "name": "my-job",
  "data": { "key": "value" }
}`
	ta.ShowLineNumbers = true
	ta.SetHeight(10)
	ta.SetWidth(60)
	ta.Focus()

	return &JobDataInput{
		textarea:  ta,
		queueName: queueName,
	}
}

func (j *JobDataInput) SetSize(width, height int) {
	j.width = width
	j.height = height
	if height > 10 {
		j.textarea.SetHeight(height - 10)
	}
}

func (j *JobDataInput) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	j.textarea, cmd = j.textarea.Update(msg)
	return cmd
}

func (j *JobDataInput) Value() string {
	return j.textarea.Value()
}

func (j *JobDataInput) SetError(err string) {
	j.err = err
}

func (j *JobDataInput) Validate() error {
	jsonStr := j.Value()

	if strings.TrimSpace(jsonStr) == "" {
		return fmt.Errorf("Job JSON cannot be empty")
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		return fmt.Errorf("Invalid JSON: %v", err)
	}

	name, nameOk := parsed["name"].(string)
	if !nameOk || name == "" {
		return fmt.Errorf("Missing or invalid 'name' field (must be string)")
	}

	if _, dataOk := parsed["data"]; !dataOk {
		return fmt.Errorf("Missing 'data' field")
	}

	return nil
}

func (j JobDataInput) View() string {
	title := ModalTitleStyle.Render(fmt.Sprintf("Add Job to Queue: %s", j.queueName))
	label := StatLabelStyle.Render("Job JSON:")

	errorMsg := ""
	if j.err != "" {
		errorStyle := StatLabelStyle.Copy().Foreground(lipgloss.Color("#EF4444"))
		errorMsg = errorStyle.Render("⚠ " + j.err)
	}

	hint := StatLabelStyle.Render("Ctrl+Enter to submit  •  Esc to cancel")

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		label,
		j.textarea.View(),
		errorMsg,
		"",
		hint,
	)

	return ModalStyle.Width(70).Render(content)
}
