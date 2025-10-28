package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/tui/views/trafficwatch"
)

type TrafficWatchTUI struct {
	recordDir     string // The directory where the recorded traffic is stored
	recordsFormat config.OutputFormat
	logsFile      string
}

func NewTrafficWatch(recordDir string, recordsFormat config.OutputFormat, logsFile string) TUI {
	return &TrafficWatchTUI{
		recordDir:     recordDir,
		recordsFormat: recordsFormat,
		logsFile:      logsFile,
	}
}

func (t *TrafficWatchTUI) Run() error {
	view, err := trafficwatch.NewMainView(t.recordDir, t.recordsFormat, t.logsFile)
	if err != nil {
		return err
	}

	p := tea.NewProgram(view, tea.WithAltScreen())
	_, err = p.Run()
	if err != nil {
		return err
	}

	return nil
}
