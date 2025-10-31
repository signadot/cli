package trafficwatch

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/signadot/cli/internal/tui/filemanager"
)

type ToggleFollowModeMsg struct {
}

func ToggleFollowMode() tea.Cmd {
	return func() tea.Msg {
		return ToggleFollowModeMsg{}
	}
}

type RefreshDataMsg struct {
	Requests []*filemanager.RequestMetadata
}

type NextPageMsg struct {
	Page int

	AutoFirst bool // When true, the selected index will not be changed
}

type PrevPageMsg struct {
	Page int

	AutoLast bool // When true, the selected index will not be changed
}

type RequestSelectedMsg struct {
	RequestID string
}

type trafficMsg struct {
	Request     *filemanager.RequestMetadata
	MessageType filemanager.MessageType
	Error       error
}
