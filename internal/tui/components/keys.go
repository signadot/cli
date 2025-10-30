package components

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
)

// KeyMap defines a set of keybindings. To work for help it must satisfy
// key.Map. It could also very easily be a map[string]key.Binding.
type KeyMap struct {
	Up         key.Binding
	Down       key.Binding
	Left       key.Binding
	Right      key.Binding
	Help       key.Binding
	Quit       key.Binding
	Refresh    key.Binding
	Logs       key.Binding
	Tab        key.Binding
	GoBack     key.Binding
	NextPage   key.Binding
	PrevPage   key.Binding
	FollowMode key.Binding

	shortHelp []key.Binding
}

type LiteralBindingName string

const (
	LiteralBindingNameUp         LiteralBindingName = "up"
	LiteralBindingNameDown       LiteralBindingName = "down"
	LiteralBindingNameLeft       LiteralBindingName = "left"
	LiteralBindingNameRight      LiteralBindingName = "right"
	LiteralBindingNameHelp       LiteralBindingName = "help"
	LiteralBindingNameQuit       LiteralBindingName = "quit"
	LiteralBindingNameRefresh    LiteralBindingName = "refresh"
	LiteralBindingNameLogs       LiteralBindingName = "logs"
	LiteralBindingNameTab        LiteralBindingName = "tab"
	LiteralBindingNameGoBack     LiteralBindingName = "go_back"
	LiteralBindingNameNextPage   LiteralBindingName = "next_page"
	LiteralBindingNamePrevPage   LiteralBindingName = "prev_page"
	LiteralBindingNameFollowMode LiteralBindingName = "follow_mode"
)

func (k *KeyMap) GetBasicShortHelpNames() []LiteralBindingName {
	return []LiteralBindingName{
		LiteralBindingNameHelp,
		LiteralBindingNameQuit,
	}
}

// SetShortHelp sets the short help using field names from the KeyMap
func (k *KeyMap) SetShortHelpByNames(fieldNames ...LiteralBindingName) {
	k.shortHelp = make([]key.Binding, 0, len(fieldNames))

	for _, name := range fieldNames {
		switch name {
		case LiteralBindingNameUp:
			k.shortHelp = append(k.shortHelp, k.Up)
		case LiteralBindingNameDown:
			k.shortHelp = append(k.shortHelp, k.Down)
		case LiteralBindingNameLeft:
			k.shortHelp = append(k.shortHelp, k.Left)
		case LiteralBindingNameRight:
			k.shortHelp = append(k.shortHelp, k.Right)
		case LiteralBindingNameHelp:
			k.shortHelp = append(k.shortHelp, k.Help)
		case LiteralBindingNameQuit:
			k.shortHelp = append(k.shortHelp, k.Quit)
		case LiteralBindingNameRefresh:
			k.shortHelp = append(k.shortHelp, k.Refresh)
		case LiteralBindingNameLogs:
			k.shortHelp = append(k.shortHelp, k.Logs)
		case LiteralBindingNameTab:
			k.shortHelp = append(k.shortHelp, k.Tab)
		case LiteralBindingNameGoBack:
			k.shortHelp = append(k.shortHelp, k.GoBack)
		case LiteralBindingNameNextPage:
			k.shortHelp = append(k.shortHelp, k.NextPage)
		case LiteralBindingNamePrevPage:
			k.shortHelp = append(k.shortHelp, k.PrevPage)
		case LiteralBindingNameFollowMode:
			k.shortHelp = append(k.shortHelp, k.FollowMode)
		}
	}
}

// ShortHelp returns keybindings to be shown in the mini help view. It's part
// of the key.Map interface.
func (k KeyMap) ShortHelp() []key.Binding {
	return k.shortHelp
}

// FullHelp returns keybindings for the expanded help view. It's part of the
// key.Map interface.
func (k KeyMap) FullHelp() [][]key.Binding {

	tmpFollowMode := k.FollowMode
	tmpFollowMode.SetHelp("f", "turn on/off follow mode")

	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right},         // first column
		{k.NextPage, k.PrevPage, tmpFollowMode}, // second column
		{k.Tab, k.Logs, k.Refresh},              // second column
		{k.Help, k.Quit},                        // third column
	}
}

var Keys = KeyMap{
	GoBack: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "go back"),
	),
	Up: key.NewBinding(
		key.WithKeys("up"),
		key.WithHelp("↑", "move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down"),
		key.WithHelp("↓", "move down"),
	),
	Left: key.NewBinding(
		key.WithKeys("left"),
		key.WithHelp("←", "move left"),
	),
	Right: key.NewBinding(
		key.WithKeys("right"),
		key.WithHelp("→", "move right"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "toggle help"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh data"),
	),
	Logs: key.NewBinding(
		key.WithKeys("l"),
		key.WithHelp("l", "toggle logs view"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "switch focus"),
	),
	NextPage: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "next page"),
	),
	PrevPage: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "previous page"),
	),
	FollowMode: key.NewBinding(
		key.WithKeys("f"),
		key.WithHelp("f", "turn on/off follow mode"),
	),
}

// NewHelpModel creates a new help model with the default keyMap
func NewHelpModel() help.Model {
	return help.New()
}
