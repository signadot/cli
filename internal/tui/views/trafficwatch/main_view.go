package trafficwatch

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/signadot/cli/internal/tui/components"
	"github.com/signadot/cli/internal/tui/models"
	"github.com/signadot/cli/internal/tui/views"
)

// MainViewState represents the current state of the main view
type MainViewState int

const (
	StateNoData MainViewState = iota
	StateWithData
	StateLogs
	StateHelp
)

// MainView represents the main traffic watch view
type MainView struct {
	state MainViewState

	dataService *models.MockDataService
	requests    []models.HTTPRequest
	selectedID  string

	leftPane  *LeftPane
	rightPane *RightPane
	logsView  *LogsView

	width    int
	height   int
	focus    string // "left" or "right"
	showHelp bool

	statusComponent *components.StatusComponent
	helpComponent   *components.HelpComponent
	help            help.Model
	keys            components.KeyMap
}

// NewMainView creates a new main view
func NewMainView() *MainView {
	dataService := models.NewMockDataService()
	requests := dataService.GetRequests()

	leftPane := NewLeftPane(requests)
	rightPane := NewRightPane()
	logsView := NewLogsView()

	statusComponent := components.NewStatusComponent(
		components.StatusSuccess,
		fmt.Sprintf("Loaded %d requests | Focus: Left Pane", len(requests)),
	)

	helpComponent := components.NewHelpComponent(
		"Traffic Watch",
		"Monitor HTTP traffic in real-time",
	)

	state := StateWithData
	if len(requests) == 0 {
		state = StateNoData
	}

	return &MainView{
		state:           state,
		dataService:     dataService,
		requests:        requests,
		leftPane:        leftPane,
		rightPane:       rightPane,
		logsView:        logsView,
		statusComponent: statusComponent,
		helpComponent:   helpComponent,
		help:            components.NewHelpModel(),
		keys:            components.Keys,
		focus:           "left",
		showHelp:        false,
	}
}

// Init initializes the main view
func (m *MainView) Init() tea.Cmd {
	return tea.Batch(
		m.leftPane.Init(),
		m.rightPane.Init(),
		m.logsView.Init(),
	)
}

// Update handles messages
func (m *MainView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		contentHeight := msg.Height - 3
		m.leftPane.SetSize(msg.Width/2, contentHeight)
		m.rightPane.SetSize(msg.Width/2, contentHeight)
		m.logsView.SetSize(msg.Width, contentHeight)
		// Set help width for proper rendering
		m.help.Width = msg.Width
		m.helpComponent.SetWidth(msg.Width)

	case tea.KeyMsg:
		// Handle help state keystrokes
		if m.state == StateHelp {
			switch {
			case key.Matches(msg, m.keys.Quit):
				return m, tea.Quit
			case key.Matches(msg, m.keys.Help), msg.String() == "esc":
				m.state = StateWithData
				m.showHelp = false
				return m, nil
			}
			// Ignore other keys in help state
			return m, nil
		}

		// Handle key bindings using the keyMap
		switch {
		case key.Matches(msg, m.keys.Help):
			m.state = StateHelp
			return m, nil
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.GoBack):
			m.state = StateWithData
			return m, nil
		case key.Matches(msg, m.keys.Logs):
			if m.state == StateLogs {
				m.state = StateWithData
				return m, nil
			} else {
				m.state = StateLogs
				return m, nil
			}
		case key.Matches(msg, m.keys.Help):
			if m.state == StateHelp {
				m.state = StateWithData
				m.showHelp = false
			} else {
				m.state = StateHelp
				m.showHelp = true
			}
			return m, nil
		case key.Matches(msg, m.keys.Refresh):
			m.refreshData()
			return m, nil
		case key.Matches(msg, m.keys.Tab):
			if m.focus == "left" {
				m.focus = "right"
				m.statusComponent.UpdateStatus(components.StatusSuccess).UpdateStatusMessage(
					fmt.Sprintf("Loaded %d requests | Focus: Right Pane", len(m.requests)),
				)
			} else {
				m.focus = "left"
				m.statusComponent.UpdateStatus(components.StatusSuccess).UpdateStatusMessage(
					fmt.Sprintf("Loaded %d requests | Focus: Left Pane", len(m.requests)),
				)
			}
			return m, nil
		case key.Matches(msg, m.keys.Right):
			// Move focus from left pane to right pane, or navigate within right pane
			if m.focus == "left" {
				m.focus = "right"
				m.statusComponent.UpdateStatus(components.StatusSuccess).UpdateStatusMessage(
					fmt.Sprintf("Loaded %d requests | Focus: Right Pane", len(m.requests)),
				)
				return m, nil
			}
			// If already on right pane, let the right pane handle tab navigation
		case key.Matches(msg, m.keys.Left):
			// Move focus from right pane to left pane, or navigate within left pane
			if m.focus == "right" {
				// Check if we're at the first tab of right pane
				if m.rightPane.GetActiveTab() == TabMeta {
					// Move focus back to left pane
					m.focus = "left"
					m.statusComponent.UpdateStatus(components.StatusSuccess).UpdateStatusMessage(
						fmt.Sprintf("Loaded %d requests | Focus: Left Pane", len(m.requests)),
					)
					return m, nil
				}
				// Otherwise, let the right pane handle tab navigation
			}
			// If already on left pane, let the left pane handle its own navigation
		}

	case RequestSelectedMsg:
		m.selectedID = msg.RequestID
		m.rightPane.SetRequest(m.dataService.GetRequestByID(msg.RequestID))
		return m, nil

	case LogsLoadedMsg:
		m.logsView.logs = msg.Logs
		return m, nil

	case views.GoToViewMsg:
		switch msg.View {
		case "main":
			m.state = StateWithData
			return m, nil
		}
	}

	// Update focused pane
	var cmd tea.Cmd
	if m.state == StateWithData {
		switch m.focus {
		case "left":
			_, cmd = m.leftPane.Update(msg)
		case "right":
			_, cmd = m.rightPane.Update(msg)
		}
	}

	if m.state == StateLogs {
		_, cmd = m.logsView.Update(msg)
	}
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// View renders the main view
func (m *MainView) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	var content strings.Builder

	switch m.state {
	case StateNoData:
		content.WriteString(m.renderNoDataState())
	case StateWithData:
		content.WriteString(m.renderWithDataState())
	case StateLogs:
		content.WriteString(m.renderLogsState())
	case StateHelp:
		content.WriteString(m.renderHelpState())
	}

	// Add help at the bottom if not in help state
	if m.state != StateHelp {
		m.statusComponent.UpdateStatusMessage(fmt.Sprintf("Loaded %d requests | Focus: %s | Help: %s", len(m.requests), m.focus, m.help.View(m.keys)))
		content.WriteString("\n")
		content.WriteString(m.statusComponent.Render())
	} else {
		content.WriteString("\n")
		m.statusComponent.UpdateStatusMessage(fmt.Sprintf("Loaded %d requests | Focus: %s", len(m.requests), m.focus))
		content.WriteString(m.statusComponent.Render())
	}

	return content.String()
}

// renderNoDataState renders the view when there's no data
func (m *MainView) renderNoDataState() string {
	emptyScreen := components.NewNoDataEmptyScreen()
	return emptyScreen.Render()
}

// renderWithDataState renders the view when there's data
func (m *MainView) renderWithDataState() string {
	leftContent := m.leftPane.View()
	rightContent := m.rightPane.View()

	paneWidth := m.width/2 - 2
	leftStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#5D95FF")).
		Padding(1).
		Width(paneWidth).
		Height(m.height - 3)

	rightStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#2E77FF")).
		Padding(1).
		Width(paneWidth).
		Height(m.height - 3)
	if m.focus == "left" {
		leftStyle = leftStyle.
			BorderForeground(lipgloss.Color("#5D95FF")).
			BorderStyle(lipgloss.ThickBorder()).
			Background(lipgloss.Color("black")).
			Foreground(lipgloss.Color("white"))
		rightStyle = rightStyle.
			BorderForeground(lipgloss.Color("gray")).
			Background(lipgloss.Color("black")).
			Foreground(lipgloss.Color("gray"))
	} else {
		rightStyle = rightStyle.
			BorderForeground(lipgloss.Color("#2E77FF")).
			BorderStyle(lipgloss.ThickBorder()).
			Background(lipgloss.Color("black")).
			Foreground(lipgloss.Color("white"))
		leftStyle = leftStyle.
			BorderForeground(lipgloss.Color("gray")).
			Background(lipgloss.Color("black")).
			Foreground(lipgloss.Color("gray"))
	}

	leftPane := leftStyle.Render(leftContent)
	rightPane := rightStyle.Render(rightContent)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
}

// renderLogsState renders the logs view
func (m *MainView) renderLogsState() string {
	return m.logsView.View()
}

// renderHelpState renders the help view
func (m *MainView) renderHelpState() string {
	return m.helpComponent.Render()
}

// refreshData refreshes the data from the service
func (m *MainView) refreshData() {
	m.requests = m.dataService.GetRequests()
	m.leftPane.SetRequests(m.requests)

	if len(m.requests) == 0 {
		m.state = StateNoData
	} else {
		m.state = StateWithData
	}

	focusText := "Left Pane"
	if m.focus == "right" {
		focusText = "Right Pane"
	}
	m.statusComponent = components.NewStatusComponent(
		components.StatusSuccess,
		fmt.Sprintf("Refreshed %d requests | Focus: %s", len(m.requests), focusText),
	)
}

// RequestSelectedMsg is sent when a request is selected
type RequestSelectedMsg struct {
	RequestID string
}
