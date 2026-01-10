package components

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
	Up            key.Binding
	Down          key.Binding
	Left          key.Binding
	Right         key.Binding
	Tab           key.Binding
	ShiftTab      key.Binding
	Enter         key.Binding
	Escape        key.Binding
	Quit          key.Binding
	Help          key.Binding
	Retry         key.Binding
	RetryAll      key.Binding
	Delete        key.Binding
	Drain         key.Binding
	Pause         key.Binding
	Refresh       key.Binding
	Filter        key.Binding
	Copy          key.Binding
	StateWaiting  key.Binding
	StateActive   key.Binding
	StateDelayed  key.Binding
	StateComplete key.Binding
	StateFailed   key.Binding
	NewQueue      key.Binding
	AddJob        key.Binding
}

var Keys = KeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Left: key.NewBinding(
		key.WithKeys("left", "h"),
		key.WithHelp("←/h", "left"),
	),
	Right: key.NewBinding(
		key.WithKeys("right", "l"),
		key.WithHelp("→/l", "right"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "switch pane"),
	),
	ShiftTab: key.NewBinding(
		key.WithKeys("shift+tab"),
		key.WithHelp("shift+tab", "switch pane back"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select/view"),
	),
	Escape: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "close/cancel"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	Retry: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "retry job"),
	),
	RetryAll: key.NewBinding(
		key.WithKeys("R"),
		key.WithHelp("R", "retry all failed"),
	),
	Delete: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "delete job"),
	),
	Drain: key.NewBinding(
		key.WithKeys("D"),
		key.WithHelp("D", "drain queue"),
	),
	Pause: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "pause/resume"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("ctrl+r"),
		key.WithHelp("ctrl+r", "refresh"),
	),
	Filter: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "filter"),
	),
	Copy: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "copy JSON"),
	),
	StateWaiting: key.NewBinding(
		key.WithKeys("1"),
		key.WithHelp("1", "waiting"),
	),
	StateActive: key.NewBinding(
		key.WithKeys("2"),
		key.WithHelp("2", "active"),
	),
	StateDelayed: key.NewBinding(
		key.WithKeys("3"),
		key.WithHelp("3", "delayed"),
	),
	StateComplete: key.NewBinding(
		key.WithKeys("4"),
		key.WithHelp("4", "completed"),
	),
	StateFailed: key.NewBinding(
		key.WithKeys("5"),
		key.WithHelp("5", "failed"),
	),
	NewQueue: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "new queue"),
	),
	AddJob: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "add job"),
	),
}
