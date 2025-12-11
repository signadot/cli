package filemanager

import (
	"bufio"
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"
)

// LogEntry represents a parsed log entry
type LogEntry struct {
	Timestamp time.Time
	Level     slog.Level
	Message   string
	Attrs     map[string]any
	RawLine   string
}

// LogFileScanner scans log files for new entries
type LogFileScanner struct {
	cfg    *LogFileScannerConfig
	offset int64

	resumeCh chan struct{}
	closeCh  chan struct{}
	closeOnce sync.Once
}

// LogFileScannerConfig holds configuration for the log file scanner
type LogFileScannerConfig struct {
	logFilePath string
	onNewLine   OnNewLogLineCallback
}

// OnNewLogLineCallback is called when a new log line is found
type OnNewLogLineCallback func(entry LogEntry)

// NewLogFileScanner creates a new log file scanner
func NewLogFileScanner(cfg *LogFileScannerConfig) *LogFileScanner {
	return &LogFileScanner{
		cfg:      cfg,
		offset:   0,
		resumeCh: make(chan struct{}),
		closeCh:  make(chan struct{}),
	}
}

// Resume resumes scanning
func (lfs *LogFileScanner) Resume() {
	select {
	case lfs.resumeCh <- struct{}{}:
	default:
	}
}

// Close closes the scanner
func (lfs *LogFileScanner) Close() {
	lfs.closeOnce.Do(func() {
		close(lfs.closeCh)
	})
}

func (lfs *LogFileScanner) Start(ctx context.Context) error {
	go lfs.monitorWithTicker(ctx)

	return nil
}

func (lfs *LogFileScanner) monitorWithTicker(ctx context.Context) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-lfs.closeCh:
			return
		case <-ticker.C:
			lfs.checkForNewContent()
		case <-lfs.resumeCh:
			lfs.checkForNewContent()
		}
	}
}

func (lfs *LogFileScanner) checkForNewContent() {
	file, err := os.Open(lfs.cfg.logFilePath)
	if err != nil {
		return
	}
	defer file.Close()

	// Seek to our last known position
	_, err = file.Seek(lfs.offset, io.SeekStart)
	if err != nil {
		return
	}

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 {
			continue
		}

		entry := parseLogLine(line)

		lfs.offset += int64(len(line)) + 1 // +1 for newline

		if lfs.cfg.onNewLine != nil {
			lfs.cfg.onNewLine(entry)
		}
	}

	if err := scanner.Err(); err != nil {
		return
	}
}

func parseTextFormat(line string) []string {
	var parts []string
	var current strings.Builder
	inQuotes := false
	escapeNext := false

	for _, r := range line {
		if escapeNext {
			current.WriteRune(r)
			escapeNext = false
			continue
		}

		if r == '\\' {
			escapeNext = true
			continue
		}

		if r == '"' {
			inQuotes = !inQuotes
			current.WriteRune(r)
			continue
		}

		if r == ' ' && !inQuotes {
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
			continue
		}

		current.WriteRune(r)
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

func parseKeyValue(part string) (key, value string) {
	eqIndex := strings.Index(part, "=")
	if eqIndex == -1 {
		return "", ""
	}

	key = part[:eqIndex]
	value = part[eqIndex+1:]

	// Remove quotes if present
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		value = value[1 : len(value)-1]
	}

	return key, value
}

func parseLogLine(line string) LogEntry {
	entry := LogEntry{
		Timestamp: time.Now(), // Default to current time
		Level:     slog.LevelInfo,
		Message:   "",
		Attrs:     make(map[string]any),
		RawLine:   line,
	}

	parts := parseTextFormat(line)

	for _, part := range parts {
		if strings.Contains(part, "=") {
			key, value := parseKeyValue(part)
			if key != "" {
				switch key {
				case "level":
					// Parse slog level
					switch strings.ToUpper(value) {
					case "DEBUG":
						entry.Level = slog.LevelDebug
					case "INFO":
						entry.Level = slog.LevelInfo
					case "WARN", "WARNING":
						entry.Level = slog.LevelWarn
					case "ERROR":
						entry.Level = slog.LevelError
					default:
						entry.Level = slog.LevelInfo
					}
				case "msg":
					entry.Message = value
				case "time":
					// Try to parse timestamp
					if t, err := time.Parse(time.RFC3339Nano, value); err == nil {
						entry.Timestamp = t
					}
				default:
					entry.Attrs[key] = value
				}
			}
		}
	}

	// If no message was found in msg field, use the whole line as message
	if entry.Message == "" {
		entry.Message = line
	}

	return entry
}

// NewLogFileScannerConfig creates a new log file scanner config
func NewLogFileScannerConfig(opts ...func(*LogFileScannerConfig)) (*LogFileScannerConfig, error) {
	cfg := &LogFileScannerConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.logFilePath == "" {
		return nil, &ConfigError{Field: "logFilePath", Message: "logFilePath is required"}
	}

	return cfg, nil
}

// WithLogFilePath sets the log file path
func WithLogFilePath(logFilePath string) func(*LogFileScannerConfig) {
	return func(config *LogFileScannerConfig) {
		config.logFilePath = logFilePath
	}
}

// WithOnNewLogLine sets the callback for new log lines
func WithOnNewLogLine(onNewLine OnNewLogLineCallback) func(*LogFileScannerConfig) {
	return func(config *LogFileScannerConfig) {
		config.onNewLine = onNewLine
	}
}

// ConfigError represents a configuration error
type ConfigError struct {
	Field   string
	Message string
}

func (e *ConfigError) Error() string {
	return e.Field + ": " + e.Message
}
