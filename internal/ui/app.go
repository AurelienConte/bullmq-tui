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
	help           *components.HelpOverlay
	toast          *components.Toast
	queueNameInput *components.QueueNameInput
	jobDataInput   *components.JobDataInput

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

func NewApp(conn *config.Connection, cfg *config.Config) *App {
	ctx, cancel := context.WithCancel(context.Background())

	return &App{
		ctx:        ctx,
		cancel:     cancel,
		conn:       conn,
		cfg:        cfg,
		focused:    focusSidebar,
		header:     components.NewHeader(conn.Name, conn.Host, conn.Port),
		sidebar:    components.NewSidebar(),
		statsPanel: components.NewStatsPanel(),
		jobTable:   components.NewJobTable(),
		statusBar:  components.NewStatusBar(),
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

		if a.queueNameInput != nil {
			return a.handleQueueNameInputKey(msg)
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

		case "n":
			// Show queue name input modal
			a.queueNameInput = components.NewQueueNameInput()
			if a.width > 0 && a.height > 0 {
				a.queueNameInput.SetSize(a.width, a.height)
			}
			return a, nil

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
	if a.queueNameInput != nil {
		view = a.overlayCenter(view, a.queueNameInput.View())
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

	// Toast (bottom right)
	if a.toast != nil {
		view = a.overlayBottomRight(view, a.toast.View())
	}

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
			a.confirm = components.NewConfirmDialog(
				"Retry Job",
				fmt.Sprintf("Retry job #%s?", job.ID),
			)
		}

	case "R":
		// Retry all failed
		a.confirm = components.NewConfirmDialog(
			"Retry All Failed",
			fmt.Sprintf("Retry all failed jobs in queue '%s'?", a.selectedQueue),
		)

	case "d":
		// Delete job
		if job := a.jobTable.SelectedJob(); job != nil {
			a.confirm = components.NewConfirmDialog(
				"Delete Job",
				fmt.Sprintf("Delete job #%s? This cannot be undone.", job.ID),
			)
		}

	case "D":
		// Drain queue
		state := a.statsPanel.GetActiveState()
		a.confirm = components.NewConfirmDialog(
			"Drain Queue",
			fmt.Sprintf("Delete ALL %s jobs in queue '%s'? This cannot be undone.", state, a.selectedQueue),
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
			// Execute the action
			// TODO: Implement actual actions based on dialog context
			a.showToast("Action not implemented yet", components.ToastInfo)
			a.confirm = nil
			return a, nil
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
			a.confirm = components.NewConfirmDialog(
				"Retry Job",
				fmt.Sprintf("Retry job #%s?", job.ID),
			)
		}

	case "d":
		// Delete from detail view
		if job := a.jobDetail.Job(); job != nil {
			a.jobDetail = nil
			a.confirm = components.NewConfirmDialog(
				"Delete Job",
				fmt.Sprintf("Delete job #%s? This cannot be undone.", job.ID),
			)
		}
	}

	return a, nil
}

func (a *App) handleQueueNameInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.queueNameInput = nil
		return a, nil

	case "enter":
		queueName := a.queueNameInput.Value()

		// Validate queue name
		if queueName == "" {
			a.queueNameInput.SetError("Queue name cannot be empty")
			return a, nil
		}

		// Check if queue already exists
		for _, q := range a.queues {
			if q.Name == queueName {
				a.queueNameInput.SetError("Queue already exists")
				return a, nil
			}
		}

		// Close modal and select the new queue
		a.queueNameInput = nil
		a.selectedQueue = queueName
		a.showToast(fmt.Sprintf("Queue '%s' ready. Add first job to create.", queueName), components.ToastInfo)

		return a, nil
	}

	// Pass other keys to the textinput component
	var cmd tea.Cmd
	if a.queueNameInput != nil {
		cmd = a.queueNameInput.Update(msg)
	}
	return a, cmd
}

func (a *App) handleJobDataInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.jobDataInput = nil
		return a, nil

	case "ctrl+enter":
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
			a.jobDataInput.SetError("InvaljobDataInputid JSON syntax")
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
		a.jobTable.SetFocused(true)
	} else {
		a.focused = focusSidebar
		a.sidebar.SetFocused(true)
		a.jobTable.SetFocused(false)
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

	a.header.SetSize(a.width)
	a.sidebar.SetSize(sidebarWidth, mainHeight)
	a.statusBar.SetSize(a.width)
	a.statsPanel.SetSize(a.width-sidebarWidth-2, 8)
	a.jobTable.SetSize(a.width-sidebarWidth-2, mainHeight-10)

	if a.help != nil {
		a.help.SetSize(a.width, a.height)
	}
	if a.confirm != nil {
		a.confirm.SetSize(a.width, a.height)
	}
	if a.jobDetail != nil {
		a.jobDetail.SetSize(a.width, a.height)
	}
	if a.queueNameInput != nil {
		a.queueNameInput.SetSize(a.width, a.height)
	}
	if a.jobDataInput != nil {
		a.jobDataInput.SetSize(a.width, a.height)
	}
}

func (a *App) showToast(message string, toastType components.ToastType) {
	a.toast = components.NewToast(message, toastType)
}

func (a *App) overlayCenter(base, overlay string) string {
	baseWidth := lipgloss.Width(base)
	baseHeight := lipgloss.Height(base)

	return lipgloss.Place(baseWidth, baseHeight, lipgloss.Center, lipgloss.Center, overlay,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("0")))
}

func (a *App) overlayBottomRight(base, overlay string) string {
	baseWidth := lipgloss.Width(base)
	baseHeight := lipgloss.Height(base)

	return lipgloss.Place(baseWidth, baseHeight, lipgloss.Right, lipgloss.Bottom, overlay)
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

// Run starts the Bubbletea program
func Run(conn *config.Connection, cfg *config.Config) error {
	app := NewApp(conn, cfg)
	p := tea.NewProgram(app, tea.WithAltScreen())

	_, err := p.Run()
	return err
}
