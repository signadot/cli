package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// HelpComponent represents a reusable help component
type HelpComponent struct {
	Title       string
	Description string
	Shortcuts   map[string]string
	keysOrder   []string
	Style       lipgloss.Style
}

// NewHelpComponent creates a new help component
func NewHelpComponent(title, description string) *HelpComponent {
	return &HelpComponent{
		Title:       title,
		Description: description,
		Shortcuts:   make(map[string]string),
		Style:       lipgloss.NewStyle().Padding(1, 2).Border(lipgloss.RoundedBorder()),
	}
}

func (h *HelpComponent) GetShortcuts() map[string]string {
	return h.Shortcuts
}

func (h *HelpComponent) GetKeysOrder() []string {
	return h.keysOrder
}

// AddShortcut adds a keyboard shortcut to the help
func (h *HelpComponent) AddShortcut(key, description string) *HelpComponent {
	h.Shortcuts[key] = description
	h.keysOrder = append(h.keysOrder, key)
	return h
}

// SetStyle allows customizing the help style
func (h *HelpComponent) SetStyle(style lipgloss.Style) *HelpComponent {
	h.Style = style
	return h
}

// Render returns the formatted help string
func (h *HelpComponent) Render() string {
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

	// Shortcuts
	if len(h.Shortcuts) > 0 {
		content.WriteString("Shortcuts:\n")
		for _, key := range h.keysOrder {
			desc := h.Shortcuts[key]
			keyStyle := lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("yellow"))
			content.WriteString(fmt.Sprintf("  %s %s\n", keyStyle.Render(key), desc))
		}
	}

	return h.Style.Render(content.String())
}

// Common help shortcuts
func (h *HelpComponent) AddCommonShortcuts() *HelpComponent {
	h.AddShortcut("q, Ctrl+C", "Quit")
	h.AddShortcut("↑/↓", "Navigate")
	h.AddShortcut("Enter", "Select")
	h.AddShortcut("Tab", "Switch focus")
	h.AddShortcut("Esc", "Go back")
	return h
}
