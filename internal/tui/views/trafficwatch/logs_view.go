package trafficwatch

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/signadot/cli/internal/tui/components"
	"github.com/signadot/cli/internal/tui/views"
)

// LogsView represents the logs view
type LogsView struct {
	logFile    string
	logs       []string
	scrollPos  int
	width      int
	height     int
	lastUpdate time.Time
}

// NewLogsView creates a new logs view
func NewLogsView() *LogsView {
	return &LogsView{
		logFile:    "testdata/traffic.log", // Default log file
		logs:       []string{},
		scrollPos:  0,
		width:      80,
		height:     20,
		lastUpdate: time.Now(),
	}
}

func (l *LogsView) Back() tea.Cmd {
	return func() tea.Msg {
		return views.GoToViewMsg{View: "main"}
	}
}

// SetSize sets the size of the logs view
func (l *LogsView) SetSize(width, height int) {
	l.width = width
	l.height = height
}

// SetLogFile sets the log file path
func (l *LogsView) SetLogFile(logFile string) {
	l.logFile = logFile
	l.loadLogs()
}

// Init initializes the logs view
func (l *LogsView) Init() tea.Cmd {
	return l.loadLogs()
}

// Update handles messages
func (l *LogsView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return l, l.Back()
		case "up", "k":
			if l.scrollPos > 0 {
				l.scrollPos--
			}
		case "down", "j":
			if l.scrollPos < len(l.logs)-l.height+2 {
				l.scrollPos++
			}
		case "g":
			l.scrollPos = 0 // Go to top
		case "G":
			l.scrollPos = len(l.logs) - l.height + 2 // Go to bottom
		case "r":
			return l, l.loadLogs()
		}
	}
	return l, nil
}

// View renders the logs view
func (l *LogsView) View() string {
	if len(l.logs) == 0 {
		return l.renderEmptyState()
	}

	var content strings.Builder

	// Header
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("blue")).
		Render(fmt.Sprintf("Logs (%d lines) - %s", len(l.logs), l.logFile))
	content.WriteString(header)
	content.WriteString("\n\n")

	// Log lines
	start := l.scrollPos
	end := start + l.height - 3
	if end > len(l.logs) {
		end = len(l.logs)
	}

	for i := start; i < end; i++ {
		line := l.logs[i]
		content.WriteString(l.renderLogLine(line))
		content.WriteString("\n")
	}

	// Scroll indicator
	if l.scrollPos > 0 || end < len(l.logs) {
		content.WriteString("\n")
		indicator := l.renderScrollIndicator()
		content.WriteString(indicator)
	}

	return content.String()
}

// renderLogLine renders a single log line with syntax highlighting
func (l *LogsView) renderLogLine(line string) string {
	// Simple log level detection
	var style lipgloss.Style

	if strings.Contains(line, "ERROR") || strings.Contains(line, "FATAL") {
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("red"))
	} else if strings.Contains(line, "WARN") {
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("yellow"))
	} else if strings.Contains(line, "INFO") {
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("blue"))
	} else if strings.Contains(line, "DEBUG") {
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("gray"))
	} else {
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("white"))
	}

	// Truncate line if too long
	if len(line) > l.width-2 {
		maxLength := l.width - 5
		if maxLength < 0 {
			maxLength = 0
		}

		if maxLength > len(line) {
			maxLength = len(line)
		}

		line = line[:maxLength] + "..."
	}

	return style.Render(line)
}

// renderScrollIndicator renders the scroll position indicator
func (l *LogsView) renderScrollIndicator() string {
	if len(l.logs) <= l.height-3 {
		return ""
	}

	progress := float64(l.scrollPos) / float64(len(l.logs)-l.height+2)
	barWidth := 20
	filled := int(progress * float64(barWidth))

	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("gray")).
		Render(fmt.Sprintf("Scroll: [%s] %d/%d", bar, l.scrollPos+1, len(l.logs)-l.height+3))
}

// renderEmptyState renders the empty state
func (l *LogsView) renderEmptyState() string {
	emptyScreen := components.NewNoLogsEmptyScreen(l.width, l.height)
	return emptyScreen.Render()
}

// loadLogs loads logs from the file
func (l *LogsView) loadLogs() tea.Cmd {
	return func() tea.Msg {
		// Try to read the log file
		file, err := os.Open(l.logFile)
		if err != nil {
			// If file doesn't exist, create some mock logs
			return LogsLoadedMsg{
				Logs: l.generateMockLogs(),
				Err:  nil,
			}
		}
		defer file.Close()

		content, err := io.ReadAll(file)
		if err != nil {
			return LogsLoadedMsg{
				Logs: l.generateMockLogs(),
				Err:  err,
			}
		}

		lines := strings.Split(string(content), "\n")
		// Filter out empty lines
		var logs []string
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				logs = append(logs, line)
			}
		}

		return LogsLoadedMsg{
			Logs: logs,
			Err:  nil,
		}
	}
}

// generateMockLogs creates some mock log entries for testing
func (l *LogsView) generateMockLogs() []string {
	now := time.Now()
	logs := []string{
		fmt.Sprintf("%s INFO  Traffic watch started", now.Format("2006-01-02 15:04:05")),
		fmt.Sprintf("%s INFO  Monitoring HTTP traffic on port 8080", now.Add(-5*time.Minute).Format("2006-01-02 15:04:05")),
		fmt.Sprintf("%s INFO  Captured request: GET /api/users", now.Add(-4*time.Minute).Format("2006-01-02 15:04:05")),
		fmt.Sprintf("%s INFO  Captured request: POST /api/users", now.Add(-3*time.Minute).Format("2006-01-02 15:04:05")),
		fmt.Sprintf("%s WARN  Slow request detected: 2.5s", now.Add(-2*time.Minute).Format("2006-01-02 15:04:05")),
		fmt.Sprintf("%s ERROR Failed to capture request: connection timeout", now.Add(-1*time.Minute).Format("2006-01-02 15:04:05")),
		fmt.Sprintf("%s INFO  Captured request: PUT /api/users/1", now.Add(-30*time.Second).Format("2006-01-02 15:04:05")),
		fmt.Sprintf("%s INFO  Captured request: DELETE /api/users/2", now.Add(-15*time.Second).Format("2006-01-02 15:04:05")),
		fmt.Sprintf("%s DEBUG Request headers: Content-Type=application/json", now.Add(-10*time.Second).Format("2006-01-02 15:04:05")),
		fmt.Sprintf("%s INFO  Captured request: GET /api/health", now.Add(-5*time.Second).Format("2006-01-02 15:04:05")),
	}

	// Add some random log entries
	for i := 0; i < 20; i++ {
		levels := []string{"INFO", "DEBUG", "WARN", "ERROR"}
		actions := []string{
			"Captured request: GET /api/orders",
			"Captured request: POST /api/auth/login",
			"Captured request: PUT /api/products/123",
			"Captured request: DELETE /api/orders/456",
			"Slow request detected: 1.8s",
			"Request completed successfully",
			"Database connection established",
			"Cache miss for key: user:123",
			"Rate limit exceeded for IP: 192.168.1.100",
			"Authentication successful for user: admin",
		}

		level := levels[i%len(levels)]
		action := actions[i%len(actions)]
		timestamp := now.Add(-time.Duration(i*10) * time.Second)

		logs = append(logs, fmt.Sprintf("%s %s %s",
			timestamp.Format("2006-01-02 15:04:05"),
			level,
			action))
	}

	return logs
}

// LogsLoadedMsg is sent when logs are loaded
type LogsLoadedMsg struct {
	Logs []string
	Err  error
}
