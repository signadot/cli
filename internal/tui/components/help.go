package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/lipgloss"
	"github.com/signadot/cli/internal/tui/colors"
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
	h.help.Width = width - 6
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
		Foreground(colors.Blue).
		Align(lipgloss.Center)

	title := titleStyle.Render(h.Title)
	content.WriteString(title)
	content.WriteString("\n\n")

	// Description
	if h.Description != "" {
		descStyle := lipgloss.NewStyle().
			Foreground(colors.LightGray).
			Align(lipgloss.Center)
		desc := descStyle.Render(h.Description)
		content.WriteString(desc)
		content.WriteString("\n\n")
	}

	content.WriteString(h.renderKeyBindings())

	view := h.Style.Render(content.String())
	h.help.ShowAll = false
	return view
}

func (h *HelpComponent) renderKeyBindings() string {
	fullHelp := h.keys.FullHelp()

	if len(fullHelp) == 0 {
		return ""
	}

	keyStyle := lipgloss.NewStyle().
		Foreground(colors.Blue).
		Bold(true).
		Padding(0, 1)

	descStyle := lipgloss.NewStyle().
		Foreground(colors.White)

	sepStyle := lipgloss.NewStyle().
		Foreground(colors.LightGray)

	var sections []string

	sectionTitles := []string{
		"Navigation",
		"Pagination",
		"View Controls",
		"General",
	}

	for i, column := range fullHelp {
		var rows []string

		if i < len(sectionTitles) {
			titleStyle := lipgloss.NewStyle().
				Bold(true).
				Foreground(colors.Blue).
				Underline(true)
			rows = append(rows, titleStyle.Render(sectionTitles[i]))
			rows = append(rows, "") // spacing
		}

		for _, binding := range column {
			if !binding.Enabled() {
				continue
			}

			keyHelp := binding.Help()
			keyStr := keyStyle.Render(keyHelp.Key)
			descStr := descStyle.Render(keyHelp.Desc)
			sep := sepStyle.Render(" • ")

			row := keyStr + sep + descStr
			rows = append(rows, row)
		}

		section := strings.Join(rows, "\n")
		sections = append(sections, section)
	}

	maxWidth := 0
	for _, section := range sections {
		lines := strings.Split(section, "\n")
		for _, line := range lines {
			width := lipgloss.Width(line)
			if width > maxWidth {
				maxWidth = width
			}
		}
	}

	sectionStyle := lipgloss.NewStyle().
		Width(maxWidth+4).
		Padding(1, 2)

	var styledSections []string
	for _, section := range sections {
		styledSections = append(styledSections, sectionStyle.Render(section))
	}

	helpContent := lipgloss.JoinHorizontal(
		lipgloss.Top,
		styledSections...,
	)

	centeredStyle := lipgloss.NewStyle().
		Width(h.help.Width).
		Align(lipgloss.Center)

	return centeredStyle.Render(helpContent)
}

// GetHelpModel returns the underlying help model for direct manipulation
func (h *HelpComponent) GetHelpModel() help.Model {
	return h.help
}
