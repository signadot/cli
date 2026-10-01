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
	width       int
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
	h.width = width
	overhead := h.getStyleOverhead()
	h.help.Width = width - overhead
}

// getStyleOverhead calculates the total width overhead from padding and borders
func (h *HelpComponent) getStyleOverhead() int {
	hPadding := h.Style.GetHorizontalPadding()
	hBorder := h.Style.GetHorizontalBorderSize()
	return hPadding + hBorder
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

	return h.Style.Render(content.String())
}

func (h *HelpComponent) renderKeyBindings() string {
	fullHelp := h.keys.FullHelp()

	if len(fullHelp) == 0 {
		return ""
	}

	// Calculate available content width
	availableWidth := h.help.Width
	if availableWidth <= 0 {
		availableWidth = 80 // default fallback
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
			rows = append(rows, "")
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

	// Calculate the width for each section
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

	// Section padding and spacing constants
	const sectionHPadding = 4 // horizontal padding per section (left + right)
	const sectionSpacing = 2  // spacing between sections
	sectionWidth := maxWidth + sectionHPadding

	// Calculate total width needed for horizontal layout
	numSections := len(sections)
	totalWidthNeeded := (sectionWidth * numSections) + (sectionSpacing * (numSections - 1))

	// Decide layout based on available width
	var helpContent string
	if totalWidthNeeded <= availableWidth && numSections > 1 {
		// Horizontal layout - sections side by side
		sectionStyle := lipgloss.NewStyle().
			Width(sectionWidth).
			Padding(1, 2)

		var styledSections []string
		for _, section := range sections {
			styledSections = append(styledSections, sectionStyle.Render(section))
		}

		helpContent = lipgloss.JoinHorizontal(
			lipgloss.Top,
			styledSections...,
		)
	} else {
		// Vertical layout - sections stacked
		// Adjust section width to use more available space
		adjustedWidth := availableWidth - sectionHPadding
		if adjustedWidth < maxWidth {
			adjustedWidth = maxWidth
		}

		sectionStyle := lipgloss.NewStyle().
			Width(adjustedWidth).
			Padding(1, 2)

		var styledSections []string
		for i, section := range sections {
			styledSections = append(styledSections, sectionStyle.Render(section))
			// Add spacing between sections except for the last one
			if i < len(sections)-1 {
				styledSections = append(styledSections, "")
			}
		}

		helpContent = lipgloss.JoinVertical(
			lipgloss.Left,
			styledSections...,
		)
	}

	// Center the entire help content
	centeredStyle := lipgloss.NewStyle().
		Width(availableWidth).
		Align(lipgloss.Center)

	return centeredStyle.Render(helpContent)
}

// GetHelpModel returns the underlying help model for direct manipulation
func (h *HelpComponent) GetHelpModel() help.Model {
	return h.help
}
