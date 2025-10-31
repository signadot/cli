package components

import (
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/signadot/cli/internal/tui/colors"
)

// StatusComponent represents a reusable status component
type StatusComponent struct {
	Status    string
	Message   string
	Timestamp time.Time
	Style     lipgloss.Style

	shortHelpMessage       string
	alwaysOnDisplayMessage string // message that is always displayed, even if the status is success
}

// NewStatusComponent creates a new status component
func NewStatusComponent(status, message string) *StatusComponent {
	return &StatusComponent{
		Status:    status,
		Message:   message,
		Timestamp: time.Now(),
		Style:     lipgloss.NewStyle().Padding(0, 1),
	}
}

func (s *StatusComponent) SetShortHelpMessage(shortHelpMessage string) *StatusComponent {
	s.shortHelpMessage = shortHelpMessage
	return s
}

func (s *StatusComponent) SetAlwaysOnDisplayMessage(alwaysOnDisplayMessage string) *StatusComponent {
	s.alwaysOnDisplayMessage = alwaysOnDisplayMessage
	return s
}

func (s *StatusComponent) UpdateStatusMessage(message string) *StatusComponent {
	s.Message = message
	return s
}

func (s *StatusComponent) UpdateStatus(status string) *StatusComponent {
	s.Status = status
	return s
}

// SetStyle allows customizing the status style
func (s *StatusComponent) SetStyle(style lipgloss.Style) *StatusComponent {
	s.Style = style
	return s
}

// Render returns the formatted status string
func (s *StatusComponent) Render(followMode bool) string {
	var content strings.Builder

	if followMode {
		followModeText := lipgloss.NewStyle().
			Foreground(colors.Blue).
			Bold(true).
			Render("[FOLLOW MODE]")
		content.WriteString(followModeText + " ")
	}

	statusColor := colors.Green
	switch s.Status {
	case "error", "failed":
		statusColor = colors.Red
	case "warning":
		statusColor = colors.Orange
	}

	statusText := lipgloss.NewStyle().
		Foreground(statusColor).
		Bold(true).
		Render(strings.ToUpper(s.Status))

	content.WriteString(statusText)
	if s.alwaysOnDisplayMessage != "" {
		content.WriteString(" " + s.alwaysOnDisplayMessage)
	} else {
		content.WriteString(" " + s.Message)
	}
	if s.shortHelpMessage != "" {
		content.WriteString("\n" + s.shortHelpMessage)
	}

	return s.Style.Render(content.String())
}

// Status types
const (
	StatusSuccess = "success"
	StatusError   = "error"
	StatusWarning = "warning"
	StatusInfo    = "info"
	StatusLoading = "loading"
)
