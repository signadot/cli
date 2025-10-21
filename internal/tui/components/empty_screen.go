package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// EmptyScreenComponent represents a reusable empty state component
type EmptyScreenComponent struct {
	Icon        string
	Title       string
	Description string
	Action      string
	Style       lipgloss.Style
}

// NewEmptyScreenComponent creates a new empty screen component
func NewEmptyScreenComponent(title, description string) *EmptyScreenComponent {
	return &EmptyScreenComponent{
		Icon:        "📭",
		Title:       title,
		Description: description,
		Style:       lipgloss.NewStyle().Align(lipgloss.Center).Padding(2),
	}
}

// SetIcon sets the icon for the empty screen
func (e *EmptyScreenComponent) SetIcon(icon string) *EmptyScreenComponent {
	e.Icon = icon
	return e
}

// SetAction sets the action text for the empty screen
func (e *EmptyScreenComponent) SetAction(action string) *EmptyScreenComponent {
	e.Action = action
	return e
}

// SetStyle allows customizing the empty screen style
func (e *EmptyScreenComponent) SetStyle(style lipgloss.Style) *EmptyScreenComponent {
	e.Style = style
	return e
}

// Render returns the formatted empty screen string
func (e *EmptyScreenComponent) Render() string {
	var content strings.Builder
	
	if e.Icon != "" {
		iconStyle := lipgloss.NewStyle().
			MarginBottom(1)
		content.WriteString(iconStyle.Render(e.Icon))
		content.WriteString("\n")
	}
	
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("blue")).
		MarginBottom(1)
	content.WriteString(titleStyle.Render(e.Title))
	content.WriteString("\n\n")
	
	if e.Description != "" {
		descStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("gray")).
			MarginBottom(1)
		content.WriteString(descStyle.Render(e.Description))
		content.WriteString("\n\n")
	}
	
	if e.Action != "" {
		actionStyle := lipgloss.NewStyle().
			Italic(true).
			Foreground(lipgloss.Color("yellow"))
		content.WriteString(actionStyle.Render(e.Action))
	}
	
	return e.Style.Render(content.String())
}

// Common empty screen types
func NewNoDataEmptyScreen() *EmptyScreenComponent {
	return NewEmptyScreenComponent(
		"No Data Available",
		"There are no HTTP requests to display at the moment.",
	).SetAction("Start monitoring traffic to see requests here.")
}

func NewNoLogsEmptyScreen() *EmptyScreenComponent {
	return NewEmptyScreenComponent(
		"No Logs Available",
		"There are no log entries to display at the moment.",
	).SetIcon("📝").SetAction("Check your log file configuration.")
}
