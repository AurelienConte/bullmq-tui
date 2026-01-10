package components

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/AurelienConte/bullmq-tui/internal/redis"
	"github.com/charmbracelet/lipgloss"
)

type JobDetail struct {
	job    *redis.Job
	width  int
	height int
	scroll int
}

func NewJobDetail(job *redis.Job) *JobDetail {
	return &JobDetail{
		job: job,
	}
}

func (j *JobDetail) SetSize(width, height int) {
	j.width = width
	j.height = height
}

func (j *JobDetail) ScrollUp() {
	if j.scroll > 0 {
		j.scroll--
	}
}

func (j *JobDetail) ScrollDown() {
	j.scroll++
}

func (j *JobDetail) Job() *redis.Job {
	return j.job
}

func (j JobDetail) View() string {
	job := j.job

	// Title
	title := ModalTitleStyle.Render(
		fmt.Sprintf("Job #%s: %s", job.ID, job.Name),
	)

	// Metadata section
	meta := []string{
		fmt.Sprintf("%-15s %s", "Status:", formatJobStatus(*job)),
		fmt.Sprintf("%-15s %s", "Created:", job.Timestamp.Format("2006-01-02 15:04:05")),
		fmt.Sprintf("%-15s %d / %d", "Attempts:", job.AttemptsMade, job.Attempts),
	}

	if job.Delay > 0 {
		meta = append(meta, fmt.Sprintf("%-15s %dms", "Delay:", job.Delay))
	}
	if job.Priority > 0 {
		meta = append(meta, fmt.Sprintf("%-15s %d", "Priority:", job.Priority))
	}
	if job.ProcessedOn != nil {
		meta = append(meta, fmt.Sprintf("%-15s %s", "Processed On:", job.ProcessedOn.Format("2006-01-02 15:04:05")))
	}
	if job.FinishedOn != nil {
		meta = append(meta, fmt.Sprintf("%-15s %s", "Finished On:", job.FinishedOn.Format("2006-01-02 15:04:05")))
	}

	// Data section
	dataSection := j.renderJSONSection("DATA", job.Data)

	// Options section
	optsSection := j.renderJSONSection("OPTIONS", job.Opts)

	// Failed reason (if failed)
	var failedSection string
	if job.State == redis.JobStateFailed && job.FailedReason != "" {
		failedSection = fmt.Sprintf(
			"\n%s\n%s",
			StatLabelStyle.Render("FAILED REASON"),
			job.FailedReason,
		)

		if len(job.Stacktrace) > 0 {
			maxLines := 5
			if len(job.Stacktrace) < maxLines {
				maxLines = len(job.Stacktrace)
			}
			failedSection += fmt.Sprintf(
				"\n\n%s\n%s",
				StatLabelStyle.Render("STACKTRACE"),
				strings.Join(job.Stacktrace[:maxLines], "\n"),
			)
		}
	}

	// Return value (if completed)
	var returnSection string
	if job.State == redis.JobStateCompleted && job.ReturnValue != "" {
		returnSection = j.renderJSONSection("RETURN VALUE", job.ReturnValue)
	}

	// Keybindings footer
	footer := StatusBarStyle.Render(
		"r retry  •  d delete  •  c copy JSON  •  Esc close",
	)

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		strings.Join(meta, "\n"),
		dataSection,
		optsSection,
		failedSection,
		returnSection,
		"",
		footer,
	)

	return ModalStyle.
		Width(min(j.width-10, 80)).
		MaxHeight(j.height - 4).
		Render(content)
}

func (j JobDetail) renderJSONSection(label, jsonStr string) string {
	if jsonStr == "" {
		return ""
	}

	header := "\n" + StatLabelStyle.Render(label)
	separator := StatLabelStyle.Render(strings.Repeat("─", 40))

	// Pretty print JSON
	var prettyJSON strings.Builder
	var data interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err == nil {
		pretty, _ := json.MarshalIndent(data, "", "  ")
		prettyJSON.Write(pretty)
	} else {
		prettyJSON.WriteString(jsonStr)
	}

	return fmt.Sprintf(
		"%s\n%s\n%s",
		header,
		separator,
		prettyJSON.String(),
	)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
