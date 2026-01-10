package components

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
)

type Header struct {
	connectionName string
	host           string
	port           int
	width          int
	connected      bool
}

func NewHeader(connName, host string, port int) Header {
	return Header{
		connectionName: connName,
		host:           host,
		port:           port,
		connected:      true,
	}
}

func (h *Header) SetSize(width int) {
	h.width = width
}

func (h *Header) SetConnected(connected bool) {
	h.connected = connected
}

func (h Header) View() string {
	title := HeaderStyle.Render("🐂 BULLMQ TUI")

	// Connection info
	connStatus := "●"
	connColor := lipgloss.Color("#10B981") // Green
	if !h.connected {
		connColor = lipgloss.Color("#EF4444") // Red
	}
	statusStyle := lipgloss.NewStyle().Foreground(connColor)

	connInfo := fmt.Sprintf("%s %s (%s:%d)",
		statusStyle.Render(connStatus),
		h.connectionName,
		h.host,
		h.port,
	)

	// Current time
	now := time.Now().Format("15:04:05")
	timeStr := StatLabelStyle.Render(now)

	// Calculate spacing
	titleLen := lipgloss.Width(title)
	connLen := lipgloss.Width(connInfo)
	timeLen := lipgloss.Width(timeStr)

	spacing1 := h.width - titleLen - connLen - timeLen - 4
	if spacing1 < 1 {
		spacing1 = 1
	}

	line := lipgloss.JoinHorizontal(
		lipgloss.Top,
		title,
		lipgloss.NewStyle().Width(spacing1).Render(""),
		connInfo,
		"  ",
		timeStr,
	)

	return lipgloss.NewStyle().
		Width(h.width).
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(HeaderStyle.GetForeground()).
		Render(line)
}
