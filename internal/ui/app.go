package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/AurelienConte/bullmq-tui/internal/config"
	"github.com/AurelienConte/bullmq-tui/internal/redis"
	"github.com/AurelienConte/bullmq-tui/internal/stats"
	"github.com/AurelienConte/bullmq-tui/internal/ui/components"
	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type focusedPanel int

const (
	focusSidebar focusedPanel = iota
	focusJobTable
)

type App struct {
	// Dependencies
	ctx       context.Context
	cancel    context.CancelFunc
	bullmq    *redis.BullMQClient
	events    *redis.EventListener
	collector *stats.Collector
	conn      *config.Connection
	cfg       *config.Config

	// UI State
	width   int
	height  int
	focused focusedPanel

	// Components
	header     components.Header
	sidebar    components.Sidebar
	statsPanel components.StatsPanel
	jobTable   components.JobTable
	statusBar  components.StatusBar

	// Overlays (nil when not shown)
	jobDetail      *components.JobDetail
	confirm        *components.ConfirmDialog
	help         *components.HelpOverlay
	toast        *components.Toast
	jobDataInput *components.JobDataInput

	// Data
	queues        []redis.Queue
	selectedQueue string
	jobs          []redis.Job
	queueStats    *stats.QueueStats

	// Error state
	err       error
	connected bool
}

type tickMsg time.Time
type queuesUpdatedMsg []redis.Queue
type jobsUpdatedMsg []redis.Job
type statsUpdatedMsg *stats.QueueStats
type eventMsg redis.JobEvent
type errMsg struct{ error }
type jobCreatedMsg string
type jobRetriedMsg struct{ ID string }
type jobDeletedMsg struct{ ID string }
type queueDrainedMsg struct{ Count int64 }
type queueCleanedMsg struct{ Count int64 }
type allFailedRetriedMsg struct{ Count int64 }

func NewApp(conn *config.Connection, cfg *config.Config) *App {
	ctx, cancel := context.WithCancel(context.Background())

	statusBar := components.NewStatusBar()
	statusBar.SetFocusedArea("Queues")

	sidebar := components.NewSidebar()
	sidebar.SetFocused(true) // Set initial focus on sidebar

	statsPanel := components.NewStatsPanel()
	statsPanel.SetFocused(false)

	jobTable := components.NewJobTable()
	jobTable.SetFocused(false)

	return &App{
		ctx:        ctx,
		cancel:     cancel,
		conn:       conn,
		cfg:        cfg,
		focused:    focusSidebar,
		header:     components.NewHeader(conn.Name, conn.Host, conn.Port),
		sidebar:    sidebar,
		statsPanel: statsPanel,
		jobTable:   jobTable,
		statusBar:  statusBar,
		connected:  false,
	}
}

func (a *App) Init() tea.Cmd {
	return tea.Batch(
		a.connectCmd(),
		a.tickCmd(),
	)
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Check if toast expired
	if a.toast != nil && a.toast.IsExpired() {
		a.toast = nil
		a.statusBar.SetMessage("") // Clear the status bar message
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle overlays first
		if a.help != nil {
			if msg.String() == "?" || msg.String() == "esc" {
				a.help = nil
			}
			return a, nil
		}

		if a.jobDataInput != nil {
			return a.handleJobDataInputKey(msg)
		}

		if a.confirm != nil {
			return a.handleConfirmKey(msg)
		}

		if a.jobDetail != nil {
			return a.handleJobDetailKey(msg)
		}

		// Global keys
		switch msg.String() {
		case "q", "ctrl+c":
			a.cancel()
			return a, tea.Quit

		case "?":
			a.help = components.NewHelpOverlay()
			return a, nil

		case "ctrl+r":
			cmds = append(cmds, a.loadQueuesCmd())
			a.showToast("Refreshing...", components.ToastInfo)

		case "tab":
			a.toggleFocus()

		case "1", "2", "3", "4", "5":
			// Switch state tabs
			oldState := a.statsPanel.GetActiveState()
			switch msg.String() {
			case "1":
				a.statsPanel.SetActiveState(redis.JobStateWaiting)
			case "2":
				a.statsPanel.SetActiveState(redis.JobStateActive)
			case "3":
				a.statsPanel.SetActiveState(redis.JobStateDelayed)
			case "4":
				a.statsPanel.SetActiveState(redis.JobStateCompleted)
			case "5":
				a.statsPanel.SetActiveState(redis.JobStateFailed)
			}
			if oldState != a.statsPanel.GetActiveState() {
				cmds = append(cmds, a.loadJobsCmd())
			}

		case "a":
			// Show job data input modal (only if queue selected)
			if a.selectedQueue == "" {
				a.showToast("No queue selected", components.ToastError)
				return a, nil
			}
			a.jobDataInput = components.NewJobDataInput(a.selectedQueue)
			if a.width > 0 && a.height > 0 {
				a.jobDataInput.SetSize(a.width, a.height)
			}
			return a, nil
		}

		// Panel-specific keys
		switch a.focused {
		case focusSidebar:
			cmds = append(cmds, a.handleSidebarKey(msg))
		case focusJobTable:
			cmds = append(cmds, a.handleJobTableKey(msg))
		}

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.updateComponentSizes()

	case tickMsg:
		cmds = append(cmds, a.tickCmd())
		if a.connected {
			cmds = append(cmds, a.loadQueuesCmd())
			// Auto-refresh job table if a queue is selected
			if a.selectedQueue != "" {
				cmds = append(cmds, a.loadJobsCmd())
			}
		}

	case queuesUpdatedMsg:
		a.queues = msg
		a.sidebar.SetQueues(msg)

		// Update stats panel counts if queue is selected
		if a.selectedQueue != "" {
			for _, q := range msg {
				if q.Name == a.selectedQueue {
					a.statsPanel.SetCounts(q.Counts)
					break
				}
			}
		}

		// Auto-select first queue if none selected
		if a.selectedQueue == "" && len(msg) > 0 {
			a.selectedQueue = msg[0].Name
			a.statsPanel.SetCounts(msg[0].Counts)
			cmds = append(cmds, a.loadJobsCmd(), a.subscribeEventsCmd())
		}

	case jobsUpdatedMsg:
		a.jobs = msg
		a.jobTable.SetJobs(msg)

	case jobCreatedMsg:
		a.showToast(fmt.Sprintf("Job #%s created successfully", string(msg)), components.ToastSuccess)
		cmds = append(cmds, a.loadJobsCmd(), a.loadQueuesCmd())

	case jobRetriedMsg:
		a.showToast(fmt.Sprintf("Job #%s retried successfully", msg.ID), components.ToastSuccess)
		cmds = append(cmds, a.loadJobsCmd(), a.loadQueuesCmd())

	case jobDeletedMsg:
		a.showToast(fmt.Sprintf("Job #%s deleted successfully", msg.ID), components.ToastSuccess)
		cmds = append(cmds, a.loadJobsCmd(), a.loadQueuesCmd())

	case allFailedRetriedMsg:
		a.showToast(fmt.Sprintf("Retried %d failed jobs", msg.Count), components.ToastSuccess)
		cmds = append(cmds, a.loadJobsCmd(), a.loadQueuesCmd())

	case queueDrainedMsg:
		a.showToast(fmt.Sprintf("Drained %d jobs from queue", msg.Count), components.ToastSuccess)
		cmds = append(cmds, a.loadJobsCmd(), a.loadQueuesCmd())

	case queueCleanedMsg:
		a.showToast(fmt.Sprintf("Cleaned %d jobs from queue", msg.Count), components.ToastSuccess)
		cmds = append(cmds, a.loadJobsCmd(), a.loadQueuesCmd())

	case statsUpdatedMsg:
		a.queueStats = msg
		a.statsPanel.SetStats(msg)

	case eventMsg:
		a.collector.HandleEvent(redis.JobEvent(msg))
		// Refresh jobs if event is for current queue and state
		if msg.QueueName == a.selectedQueue {
			cmds = append(cmds, a.loadJobsCmd())
		}

	case errMsg:
		a.err = msg.error
		a.connected = false
		a.header.SetConnected(false)
		a.showToast(fmt.Sprintf("Error: %v", msg.error), components.ToastError)
	}

	return a, tea.Batch(cmds...)
}

func (a *App) View() string {
	if a.width == 0 {
		return "Loading..."
	}

	// Build main layout
	sidebar := a.sidebar.View()
	statsPanel := a.statsPanel.View()
	jobTable := a.jobTable.View()

	rightPanel := lipgloss.JoinVertical(
		lipgloss.Left,
		statsPanel,
		jobTable,
	)

	main := lipgloss.JoinHorizontal(
		lipgloss.Top,
		sidebar,
		rightPanel,
	)

	view := lipgloss.JoinVertical(
		lipgloss.Left,
		a.header.View(),
		main,
		a.statusBar.View(),
	)

	// Overlay modals (centered)
	if a.help != nil {
		view = a.overlayCenter(view, a.help.View())
	}
	if a.jobDataInput != nil {
		view = a.overlayCenter(view, a.jobDataInput.View())
	}
	if a.confirm != nil {
		view = a.overlayCenter(view, a.confirm.View())
	}
	if a.jobDetail != nil {
		view = a.overlayCenter(view, a.jobDetail.View())
	}

	// Toast messages are now shown in the status bar instead of as an overlay
	// to avoid the white screen flash issue

	return view
}

func (a *App) handleSidebarKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "up", "k":
		a.sidebar.MoveUp()
		return a.onQueueChanged()

	case "down", "j":
		a.sidebar.MoveDown()
		return a.onQueueChanged()

	case "enter":
		return a.onQueueChanged()

	case "p":
		// Pause/resume queue
		if a.selectedQueue != "" {
			a.confirm = components.NewConfirmDialog(
				"Pause/Resume Queue",
				fmt.Sprintf("Pause or resume queue '%s'?", a.selectedQueue),
			)
		}

	case "D":
		// Clean all jobs from all states in the queue
		if a.selectedQueue != "" {
			a.confirm = components.NewConfirmDialogWithAction(
				"Clean Queue",
				fmt.Sprintf("Clean ALL jobs (all states) in queue '%s'? This cannot be undone.", a.selectedQueue),
				components.ConfirmActionCleanQueue,
				a.selectedQueue,
				"",
				redis.JobStateWaiting, // State doesn't matter for CleanQueue
			)
		}
	}

	return nil
}

func (a *App) handleJobTableKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "up", "k":
		a.jobTable.MoveUp()

	case "down", "j":
		a.jobTable.MoveDown()

	case "left", "h":
		a.statsPanel.PreviousTab()
		return a.loadJobsCmd()

	case "right", "l":
		a.statsPanel.NextTab()
		return a.loadJobsCmd()

	case "enter":
		// Show job detail
		if job := a.jobTable.SelectedJob(); job != nil {
			a.jobDetail = components.NewJobDetail(job)
		}

	case "r":
		// Retry job
		if job := a.jobTable.SelectedJob(); job != nil {
			a.confirm = components.NewConfirmDialogWithAction(
				"Retry Job",
				fmt.Sprintf("Retry job #%s?", job.ID),
				components.ConfirmActionRetryJob,
				job.QueueName,
				job.ID,
				job.State,
			)
		}

	case "R":
		// Retry all failed
		a.confirm = components.NewConfirmDialogWithAction(
			"Retry All Failed",
			fmt.Sprintf("Retry all failed jobs in queue '%s'?", a.selectedQueue),
			components.ConfirmActionRetryAllFailed,
			a.selectedQueue,
			"",
			redis.JobStateFailed,
		)

	case "d":
		// Delete job
		if job := a.jobTable.SelectedJob(); job != nil {
			a.confirm = components.NewConfirmDialogWithAction(
				"Delete Job",
				fmt.Sprintf("Delete job #%s? This cannot be undone.", job.ID),
				components.ConfirmActionDeleteJob,
				job.QueueName,
				job.ID,
				job.State,
			)
		}

	case "D":
		// Drain queue (all jobs in current state)
		state := a.statsPanel.GetActiveState()
		a.confirm = components.NewConfirmDialogWithAction(
			"Drain Queue",
			fmt.Sprintf("Delete ALL %s jobs in queue '%s'? This cannot be undone.", state, a.selectedQueue),
			components.ConfirmActionDrainQueue,
			a.selectedQueue,
			"",
			state,
		)
	}

	return nil
}

func (a *App) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.confirm = nil
		return a, nil

	case "left", "h", "right", "l":
		a.confirm.ToggleSelection()
		return a, nil

	case "enter":
		if a.confirm.IsYesSelected() {
			// Execute the action based on dialog context
			var cmd tea.Cmd
			switch a.confirm.GetAction() {
			case components.ConfirmActionRetryJob:
				cmd = a.retryJobCmd(a.confirm.GetQueueName(), a.confirm.GetJobID())
			case components.ConfirmActionDeleteJob:
				cmd = a.deleteJobCmd(a.confirm.GetQueueName(), a.confirm.GetJobID(), a.confirm.GetJobState())
			case components.ConfirmActionRetryAllFailed:
				cmd = a.retryAllFailedCmd(a.confirm.GetQueueName())
			case components.ConfirmActionDrainQueue:
				cmd = a.drainQueueCmd(a.confirm.GetQueueName(), a.confirm.GetJobState())
			case components.ConfirmActionCleanQueue:
				cmd = a.cleanQueueCmd(a.confirm.GetQueueName())
			}
			a.confirm = nil
			return a, cmd
		}
		a.confirm = nil
		return a, nil
	}

	return a, nil
}

func (a *App) handleJobDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.jobDetail = nil
		return a, nil

	case "up", "k":
		a.jobDetail.ScrollUp()

	case "down", "j":
		a.jobDetail.ScrollDown()

	case "r":
		// Retry from detail view
		if job := a.jobDetail.Job(); job != nil {
			a.jobDetail = nil
			a.confirm = components.NewConfirmDialogWithAction(
				"Retry Job",
				fmt.Sprintf("Retry job #%s?", job.ID),
				components.ConfirmActionRetryJob,
				job.QueueName,
				job.ID,
				job.State,
			)
		}

	case "d":
		// Delete from detail view
		if job := a.jobDetail.Job(); job != nil {
			a.jobDetail = nil
			a.confirm = components.NewConfirmDialogWithAction(
				"Delete Job",
				fmt.Sprintf("Delete job #%s? This cannot be undone.", job.ID),
				components.ConfirmActionDeleteJob,
				job.QueueName,
				job.ID,
				job.State,
			)
		}

	case "c":
		// Copy job JSON to clipboard
		if job := a.jobDetail.Job(); job != nil {
			// Marshal job to JSON
			jsonBytes, err := json.MarshalIndent(job, "", "  ")
			if err != nil {
				a.showToast(fmt.Sprintf("Failed to marshal JSON: %v", err), components.ToastError)
				return a, nil
			}

			// Copy to clipboard
			if err := clipboard.WriteAll(string(jsonBytes)); err != nil {
				a.showToast(fmt.Sprintf("Failed to copy to clipboard: %v", err), components.ToastError)
			} else {
				a.showToast("Job JSON copied to clipboard!", components.ToastSuccess)
			}
		}
	}

	return a, nil
}

func (a *App) handleJobDataInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Check for submission keys first, before passing to textarea
	keyStr := msg.String()

	// Handle esc
	if keyStr == "esc" {
		a.jobDataInput = nil
		return a, nil
	}

	// Handle submission with alt+enter
	isSubmit := keyStr == "alt+enter" || keyStr == "alt+return"

	if isSubmit {
		jsonStr := a.jobDataInput.Value()

		// Validate JSON
		if err := a.jobDataInput.Validate(); err != nil {
			a.jobDataInput.SetError(err.Error())
			return a, nil
		}

		// Parse JSON
		var jobInput struct {
			Name string                 `json:"name"`
			Data map[string]interface{} `json:"data"`
			Opts map[string]interface{} `json:"opts"`
		}

		if err := json.Unmarshal([]byte(jsonStr), &jobInput); err != nil {
			a.jobDataInput.SetError("Invalid JSON syntax")
			return a, nil
		}

		// Close modal and create job
		a.jobDataInput = nil
		return a, a.createJobCmd(a.selectedQueue, jobInput.Name, jobInput.Data, jobInput.Opts)
	}

	// Pass other keys to the textarea component
	var cmd tea.Cmd
	if a.jobDataInput != nil {
		cmd = a.jobDataInput.Update(msg)
	}
	return a, cmd
}

// Helpers
func (a *App) toggleFocus() {
	if a.focused == focusSidebar {
		a.focused = focusJobTable
		a.sidebar.SetFocused(false)
		a.statsPanel.SetFocused(true)
		a.jobTable.SetFocused(true)
		a.statusBar.SetFocusedArea("Jobs")
	} else {
		a.focused = focusSidebar
		a.sidebar.SetFocused(true)
		a.statsPanel.SetFocused(false)
		a.jobTable.SetFocused(false)
		a.statusBar.SetFocusedArea("Queues")
	}
}

func (a *App) onQueueChanged() tea.Cmd {
	newQueue := a.sidebar.SelectedQueue()
	if newQueue != "" && newQueue != a.selectedQueue {
		a.selectedQueue = newQueue

		// Update counts
		for _, q := range a.queues {
			if q.Name == newQueue {
				a.statsPanel.SetCounts(q.Counts)
				break
			}
		}

		return tea.Batch(a.loadJobsCmd(), a.updateStatsCmd())
	}
	return nil
}

func (a *App) updateComponentSizes() {
	headerHeight := 3
	statusBarHeight := 3
	sidebarWidth := 35
	mainHeight := a.height - headerHeight - statusBarHeight

	statsPanelHeight := 8

	a.header.SetSize(a.width)
	a.sidebar.SetSize(sidebarWidth, mainHeight)
	a.statusBar.SetSize(a.width)
	a.statsPanel.SetSize(a.width-sidebarWidth-2, statsPanelHeight)
	a.jobTable.SetSize(a.width-sidebarWidth-2, mainHeight-statsPanelHeight)

	if a.help != nil {
		a.help.SetSize(a.width, a.height)
	}
	if a.confirm != nil {
		a.confirm.SetSize(a.width, a.height)
	}
	if a.jobDetail != nil {
		a.jobDetail.SetSize(a.width, a.height)
	}
	if a.jobDataInput != nil {
		a.jobDataInput.SetSize(a.width, a.height)
	}
}

func (a *App) showToast(message string, toastType components.ToastType) {
	// Format message with icon based on type
	var formattedMessage string
	switch toastType {
	case components.ToastSuccess:
		formattedMessage = "✓ " + message
	case components.ToastError:
		formattedMessage = "✗ " + message
	default:
		formattedMessage = "ℹ " + message
	}

	// Set message in status bar
	a.statusBar.SetMessage(formattedMessage)

	// Also keep the toast for the overlay (as fallback)
	a.toast = components.NewToast(message, toastType)
}

func (a *App) overlayCenter(base, overlay string) string {
	baseWidth := lipgloss.Width(base)
	baseHeight := lipgloss.Height(base)

	return lipgloss.Place(baseWidth, baseHeight, lipgloss.Center, lipgloss.Center, overlay,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("0")))
}

// Command generators
func (a *App) tickCmd() tea.Cmd {
	interval := time.Duration(a.cfg.Settings.RefreshIntervalMs) * time.Millisecond
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (a *App) connectCmd() tea.Cmd {
	return func() tea.Msg {
		// Create Redis client
		client, err := redis.NewClient(a.ctx, a.conn)
		if err != nil {
			return errMsg{err}
		}

		a.bullmq = redis.NewBullMQClient(client, a.conn.Prefix)
		a.events = redis.NewEventListener(client, a.conn.Prefix)
		a.collector = stats.NewCollector(a.cfg.Settings.StatsWindowMinutes)
		a.connected = true
		a.header.SetConnected(true)

		return a.loadQueuesCmd()()
	}
}

func (a *App) loadQueuesCmd() tea.Cmd {
	if !a.connected || a.bullmq == nil {
		return nil
	}

	return func() tea.Msg {
		names, err := a.bullmq.DiscoverQueues(a.ctx)
		if err != nil {
			return errMsg{err}
		}

		var result []redis.Queue
		for _, name := range names {
			counts, _ := a.bullmq.GetQueueCounts(a.ctx, name)
			paused, _ := a.bullmq.IsQueuePaused(a.ctx, name)

			failureRate := float64(0)
			if counts.Completed+counts.Failed > 0 {
				failureRate = float64(counts.Failed) / float64(counts.Completed+counts.Failed) * 100
			}

			result = append(result, redis.Queue{
				Name:        name,
				IsPaused:    paused,
				Counts:      *counts,
				FailureRate: failureRate,
				HasFailures: counts.Failed > 0,
			})
		}

		return queuesUpdatedMsg(result)
	}
}

func (a *App) loadJobsCmd() tea.Cmd {
	if !a.connected || a.bullmq == nil || a.selectedQueue == "" {
		return nil
	}

	return func() tea.Msg {
		state := a.statsPanel.GetActiveState()
		jobs, err := a.bullmq.GetJobs(a.ctx, a.selectedQueue, state, 0, int64(a.cfg.Settings.MaxJobsDisplay-1))
		if err != nil {
			return errMsg{err}
		}
		return jobsUpdatedMsg(jobs)
	}
}

func (a *App) updateStatsCmd() tea.Cmd {
	if a.selectedQueue == "" {
		return nil
	}

	return func() tea.Msg {
		stats := a.collector.GetStats(a.selectedQueue)
		return statsUpdatedMsg(stats)
	}
}

func (a *App) subscribeEventsCmd() tea.Cmd {
	if !a.connected || a.events == nil {
		return nil
	}

	return func() tea.Msg {
		queueNames := make([]string, len(a.queues))
		for i, q := range a.queues {
			queueNames[i] = q.Name
		}

		eventChan, err := a.events.Subscribe(a.ctx, queueNames)
		if err != nil {
			return errMsg{err}
		}

		// Listen for events in background
		go func() {
			for event := range eventChan {
				// Send event to Bubbletea (need to use program.Send)
				_ = event
			}
		}()

		return nil
	}
}

func (a *App) createJobCmd(queueName, jobName string, jobData, opts map[string]interface{}) tea.Cmd {
	if !a.connected || a.bullmq == nil {
		return nil
	}

	return func() tea.Msg {
		jobID, err := a.bullmq.AddJob(a.ctx, queueName, jobName, jobData, opts)
		if err != nil {
			return errMsg{fmt.Errorf("failed to create job: %w", err)}
		}
		return jobCreatedMsg(jobID)
	}
}

func (a *App) retryJobCmd(queueName, jobID string) tea.Cmd {
	if !a.connected || a.bullmq == nil {
		return nil
	}

	return func() tea.Msg {
		err := a.bullmq.RetryJob(a.ctx, queueName, jobID)
		if err != nil {
			return errMsg{fmt.Errorf("failed to retry job: %w", err)}
		}
		return jobRetriedMsg{ID: jobID}
	}
}

func (a *App) deleteJobCmd(queueName, jobID string, state redis.JobState) tea.Cmd {
	if !a.connected || a.bullmq == nil {
		return nil
	}

	return func() tea.Msg {
		err := a.bullmq.DeleteJob(a.ctx, queueName, jobID, state)
		if err != nil {
			return errMsg{fmt.Errorf("failed to delete job: %w", err)}
		}
		return jobDeletedMsg{ID: jobID}
	}
}

func (a *App) retryAllFailedCmd(queueName string) tea.Cmd {
	if !a.connected || a.bullmq == nil {
		return nil
	}

	return func() tea.Msg {
		count, err := a.bullmq.RetryAllFailed(a.ctx, queueName)
		if err != nil {
			return errMsg{fmt.Errorf("failed to retry all failed jobs: %w", err)}
		}
		return allFailedRetriedMsg{Count: count}
	}
}

func (a *App) drainQueueCmd(queueName string, state redis.JobState) tea.Cmd {
	if !a.connected || a.bullmq == nil {
		return nil
	}

	return func() tea.Msg {
		count, err := a.bullmq.DrainQueue(a.ctx, queueName, state)
		if err != nil {
			return errMsg{fmt.Errorf("failed to drain queue: %w", err)}
		}
		return queueDrainedMsg{Count: count}
	}
}

func (a *App) cleanQueueCmd(queueName string) tea.Cmd {
	if !a.connected || a.bullmq == nil {
		return nil
	}

	return func() tea.Msg {
		count, err := a.bullmq.CleanQueue(a.ctx, queueName)
		if err != nil {
			return errMsg{fmt.Errorf("failed to clean queue: %w", err)}
		}
		return queueCleanedMsg{Count: count}
	}
}

// Run starts the Bubbletea program
func Run(conn *config.Connection, cfg *config.Config) error {
	app := NewApp(conn, cfg)
	p := tea.NewProgram(app, tea.WithAltScreen())

	_, err := p.Run()
	return err
}
