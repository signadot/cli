package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/lipgloss"
)

// HelpComponent represents a reusable help component using bubbles help
type HelpComponent struct {
	Title       string
	Description string
	help        help.Model
	keys        KeyMap
	Style       lipgloss.Style
}

// NewHelpComponent creates a new help component
func NewHelpComponent(title, description string) *HelpComponent {
	return &HelpComponent{
		Title:       title,
		Description: description,
		help:        NewHelpModel(),
		keys:        Keys,
		Style:       lipgloss.NewStyle().Padding(1, 2).Border(lipgloss.RoundedBorder()),
	}
}

// SetWidth sets the width of the help component
func (h *HelpComponent) SetWidth(width int) {
	h.help.Width = width
}

// ShowAll toggles the help view between short and full
func (h *HelpComponent) ShowAll(show bool) {
	h.help.ShowAll = show
}

// ToggleHelp toggles the help view
func (h *HelpComponent) ToggleHelp() {
	h.help.ShowAll = !h.help.ShowAll
}

// IsShowingAll returns whether the full help is being shown
func (h *HelpComponent) IsShowingAll() bool {
	return h.help.ShowAll
}

// Render returns the formatted help string
func (h *HelpComponent) Render() string {
	h.help.ShowAll = true

	var content strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("blue"))
	content.WriteString(titleStyle.Render(h.Title))
	content.WriteString("\n\n")

	// Description
	if h.Description != "" {
		content.WriteString(h.Description)
		content.WriteString("\n\n")
	}

	// Use bubbles help to render the key bindings
	helpView := h.help.View(h.keys)
	content.WriteString(helpView)

	view := h.Style.Render(content.String())
	h.help.ShowAll = false
	return view
}

// GetHelpModel returns the underlying help model for direct manipulation
func (h *HelpComponent) GetHelpModel() help.Model {
	return h.help
}
