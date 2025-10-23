package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// StatusComponent represents a reusable status component
type StatusComponent struct {
	Status    string
	Message   string
	Timestamp time.Time
	Style     lipgloss.Style
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
func (s *StatusComponent) Render() string {
	statusColor := lipgloss.Color("green")
	switch s.Status {
	case "error", "failed":
		statusColor = lipgloss.Color("red")
	case "warning":
		statusColor = lipgloss.Color("yellow")
	}

	statusText := lipgloss.NewStyle().
		Foreground(statusColor).
		Bold(true).
		Render(strings.ToUpper(s.Status))

	content := fmt.Sprintf("%s %s", statusText, s.Message)
	return s.Style.Render(content)
}

// Status types
const (
	StatusSuccess = "success"
	StatusError   = "error"
	StatusWarning = "warning"
	StatusInfo    = "info"
	StatusLoading = "loading"
)
