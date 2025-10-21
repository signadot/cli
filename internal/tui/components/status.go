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

// SetStyle allows customizing the status style
func (s *StatusComponent) SetStyle(style lipgloss.Style) *StatusComponent {
	s.Style = style
	return s
}

// Render returns the formatted status string
func (s *StatusComponent) Render() string {
	statusColor := lipgloss.Color("green")
	if s.Status == "error" || s.Status == "failed" {
		statusColor = lipgloss.Color("red")
	} else if s.Status == "warning" {
		statusColor = lipgloss.Color("yellow")
	}

	statusText := lipgloss.NewStyle().
		Foreground(statusColor).
		Bold(true).
		Render(strings.ToUpper(s.Status))

	timestamp := s.Timestamp.Format("15:04:05")
	
	content := fmt.Sprintf("%s %s [%s]", statusText, s.Message, timestamp)
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
