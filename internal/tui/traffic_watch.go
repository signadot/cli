package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/signadot/cli/internal/tui/views/trafficwatch"
)

type TrafficWatchTUI struct {
	RecordDir string // The directory where the recorded traffic is stored
}

func NewTrafficWatch(recordDir string) TUI {
	return &TrafficWatchTUI{
		RecordDir: recordDir,
	}
}

func (t *TrafficWatchTUI) Run() error {
	view := trafficwatch.NewMainView()

	p := tea.NewProgram(view, tea.WithAltScreen())
	_, err := p.Run()
	if err != nil {
		return err
	}

	return nil
}
