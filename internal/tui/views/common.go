package views

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

type CommonViewModel struct {
	quitting bool
}

type GoToViewMsg struct {
	View string
}

func NewCommonViewModel() *CommonViewModel {
	return &CommonViewModel{}
}

func (m *CommonViewModel) Init() tea.Cmd {
	return nil
}

func (m *CommonViewModel) Update(msg tea.Msg) (*CommonViewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Batch(tea.Quit, tea.ExitAltScreen)
		}
	}

	if m.quitting {
		fmt.Println("Quitting...")
		return m, tea.Batch(tea.Quit, tea.ExitAltScreen)
	}

	return m, nil
}

func (m *CommonViewModel) View() string {
	if m.quitting {
		return "Quitting..."
	}

	return ""
}
