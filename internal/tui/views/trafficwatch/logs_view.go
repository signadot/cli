package trafficwatch

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/signadot/cli/internal/tui/components"
	"github.com/signadot/cli/internal/tui/filemanager"
	"github.com/signadot/cli/internal/tui/views"
)

var OrderArr = []string{
	"level",
	"msg",
	"time",
	"sandbox",
	"request.id",
	"request.dest",
	"request.uri",
	"request.method",
	"request.userAgent",
	"request.doneAt",
}

// LogsView represents the logs view
type LogsView struct {
	logFile    string
	logs       []filemanager.LogEntry
	width      int
	height     int
	lastUpdate time.Time
	scanner    *filemanager.LogFileScanner
	ctx        context.Context
	cancel     context.CancelFunc
	viewport   viewport.Model
	ready      bool
}

// NewLogsView creates a new logs view
func NewLogsView(logsFile string) *LogsView {
	ctx, cancel := context.WithCancel(context.Background())
	return &LogsView{
		logFile:    logsFile,
		logs:       []filemanager.LogEntry{},
		width:      80,
		height:     20,
		lastUpdate: time.Now(),
		ctx:        ctx,
		cancel:     cancel,
		ready:      false,
	}
}

func (l *LogsView) Back() tea.Cmd {
	l.cleanup()
	return func() tea.Msg {
		return views.GoToViewMsg{View: "main"}
	}
}

// cleanup cleans up resources
func (l *LogsView) cleanup() {
	if l.scanner != nil {
		l.scanner.Close()
	}
	if l.cancel != nil {
		l.cancel()
	}
}

// SetSize sets the size of the logs view
func (l *LogsView) SetSize(width, height int) {
	l.width = width
	l.height = height

	// Initialize viewport if not ready
	if !l.ready {
		headerHeight := lipgloss.Height(l.headerView())
		footerHeight := lipgloss.Height(l.footerView())
		verticalMarginHeight := headerHeight + footerHeight

		l.viewport = viewport.New(width, height-verticalMarginHeight)
		l.viewport.YPosition = headerHeight
		l.viewport.SetContent(l.buildContent())
		l.ready = true
	} else {
		// Update existing viewport size
		headerHeight := lipgloss.Height(l.headerView())
		footerHeight := lipgloss.Height(l.footerView())
		verticalMarginHeight := headerHeight + footerHeight

		l.viewport.Width = width
		l.viewport.Height = height - verticalMarginHeight
	}
}

// SetLogFile sets the log file path
func (l *LogsView) SetLogFile(logFile string) {
	l.logFile = logFile
	l.initializeScanner()
	l.loadLogs()
}

// Init initializes the logs view
func (l *LogsView) Init() tea.Cmd {
	l.initializeScanner()
	return l.loadLogs()
}

// initializeScanner initializes the log file scanner
func (l *LogsView) initializeScanner() {
	if l.scanner != nil {
		l.scanner.Close()
	}

	cfg, err := filemanager.NewLogFileScannerConfig(
		filemanager.WithLogFilePath(l.logFile),
		filemanager.WithOnNewLogLine(l.onNewLogEntry),
	)
	if err != nil {
		return
	}

	l.scanner = filemanager.NewLogFileScanner(cfg)
	go l.scanner.Start(l.ctx)
}

// onNewLogEntry handles new log entries from the scanner
func (l *LogsView) onNewLogEntry(entry filemanager.LogEntry) {
	l.logs = append(l.logs, entry)
	l.lastUpdate = time.Now()

	// Update viewport content if ready
	if l.ready {
		l.viewport.SetContent(l.buildContent())
		// Auto-scroll to bottom for new logs
		l.viewport.GotoBottom()
	}
}

// Update handles messages
func (l *LogsView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return l, l.Back()
		case "r":
			return l, l.loadLogs()
		}
	case tea.WindowSizeMsg:
		// Update viewport size if already initialized
		if l.ready {
			headerHeight := lipgloss.Height(l.headerView())
			footerHeight := lipgloss.Height(l.footerView())
			verticalMarginHeight := headerHeight + footerHeight

			l.viewport.Width = msg.Width
			l.viewport.Height = msg.Height - verticalMarginHeight
		}
	case LogsLoadedMsg:
		l.logs = msg.Logs
		if l.ready {
			l.viewport.SetContent(l.buildContent())
			// Auto-scroll to bottom for new logs
			l.viewport.GotoBottom()
		}
	}

	// Handle keyboard and mouse events in the viewport
	if l.ready {
		l.viewport, cmd = l.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return l, tea.Batch(cmds...)
}

// View renders the logs view
func (l *LogsView) View() string {
	if len(l.logs) == 0 {
		return l.renderEmptyState()
	}

	if !l.ready {
		// Show a simple loading state instead of "Initializing..."
		return fmt.Sprintf("%s\n\n%s", l.headerView(), "Loading logs...")
	}

	return fmt.Sprintf("%s\n%s\n%s", l.headerView(), l.viewport.View(), l.footerView())
}

// headerView renders the header
func (l *LogsView) headerView() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("blue")).
		Render(fmt.Sprintf("Logs (%d entries) - %s", len(l.logs), l.logFile))

	if l.ready {
		line := strings.Repeat("─", max(0, l.viewport.Width-lipgloss.Width(title)))
		return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
	}
	return title
}

// footerView renders the footer with scroll percentage
func (l *LogsView) footerView() string {
	if !l.ready {
		return ""
	}

	info := lipgloss.NewStyle().
		Foreground(lipgloss.Color("gray")).
		Render(fmt.Sprintf("%3.f%%", l.viewport.ScrollPercent()*100))

	line := strings.Repeat("─", max(0, l.viewport.Width-lipgloss.Width(info)))
	return lipgloss.JoinHorizontal(lipgloss.Center, line, info)
}

// buildContent builds the content for the viewport
func (l *LogsView) buildContent() string {
	if len(l.logs) == 0 {
		return "No logs available"
	}

	var content strings.Builder
	for _, entry := range l.logs {
		content.WriteString(l.renderLogLine(entry))
		content.WriteString("\n")
	}

	return content.String()
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// renderLogLine renders a single log line with syntax highlighting
func (l *LogsView) renderLogLine(entry filemanager.LogEntry) string {
	// Create a styled timestamp
	timestamp := entry.Timestamp.Format("15:04:05")
	timestampStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("gray"))

	// Create level styling based on slog.Level
	var levelStyle lipgloss.Style
	switch entry.Level {
	case slog.LevelError:
		levelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("red")).Bold(true)
	case slog.LevelWarn:
		levelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("yellow")).Bold(true)
	case slog.LevelInfo:
		levelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("blue"))
	case slog.LevelDebug:
		levelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("gray"))
	default:
		levelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("white"))
	}

	// Format the log line
	var line strings.Builder

	// Add timestamp
	line.WriteString(timestampStyle.Render(timestamp))
	line.WriteString(" ")

	// Add level
	levelText := fmt.Sprintf("[%s]", entry.Level.String())
	line.WriteString(levelStyle.Render(levelText))
	line.WriteString(" ")

	// Add message
	messageStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("white"))
	line.WriteString(messageStyle.Render(entry.Message))

	// Add all attributes with different colors for different types
	for _, key := range OrderArr {
		value, ok := entry.Attrs[key]
		if !ok {
			continue
		}
		var attrStyle lipgloss.Style

		// Color-code different types of attributes
		switch {
		case strings.HasPrefix(key, "request."):
			attrStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("green"))
		case key == "sandbox":
			attrStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("cyan"))
		case key == "error":
			attrStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("red"))
		case strings.Contains(key, "time") || strings.Contains(key, "At"):
			attrStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("magenta"))
		case key == "level" || key == "msg":
			// Skip these as they're already displayed
			continue
		default:
			attrStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("yellow"))
		}

		line.WriteString(" ")
		line.WriteString(attrStyle.Render(fmt.Sprintf("%s=%v", key, value)))
	}

	return lipgloss.NewStyle().SetString(line.String()).Width(l.viewport.Width - 2).Render()
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
				Err: nil,
			}
		}
		defer file.Close()

		content, err := io.ReadAll(file)
		if err != nil {
			return LogsLoadedMsg{
				Err: err,
			}
		}

		lines := strings.Split(string(content), "\n")
		var logs []filemanager.LogEntry

		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				entry := filemanager.ParseLogLine(line)
				logs = append(logs, entry)
			}
		}

		return LogsLoadedMsg{
			Logs: logs,
			Err:  nil,
		}
	}
}

// LogsLoadedMsg is sent when logs are loaded
type LogsLoadedMsg struct {
	Logs []filemanager.LogEntry
	Err  error
}
