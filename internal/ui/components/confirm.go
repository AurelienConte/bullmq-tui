package components

import (
	"github.com/charmbracelet/lipgloss"
)

type ConfirmDialog struct {
	title    string
	message  string
	selected int // 0 = Yes, 1 = No
	width    int
	height   int
}

func NewConfirmDialog(title, message string) *ConfirmDialog {
	return &ConfirmDialog{
		title:    title,
		message:  message,
		selected: 1, // Default to No
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
