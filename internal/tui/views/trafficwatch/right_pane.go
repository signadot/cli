package trafficwatch

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/signadot/cli/internal/tui/components"
	"github.com/signadot/cli/internal/tui/models"
)

// RightPaneTab represents the active tab
type RightPaneTab int

const (
	TabMeta RightPaneTab = iota
	TabRequest
	TabResponse
)

// RightPane represents the right pane showing request details
type RightPane struct {
	request   *models.HTTPRequest
	activeTab RightPaneTab
	width     int
	height    int
}

// NewRightPane creates a new right pane
func NewRightPane() *RightPane {
	return &RightPane{
		activeTab: TabMeta,
		width:     50,
		height:    20,
	}
}

// SetSize sets the size of the right pane
func (r *RightPane) SetSize(width, height int) {
	r.width = width
	r.height = height
}

// SetRequest sets the current request to display
func (r *RightPane) SetRequest(request *models.HTTPRequest) {
	r.request = request
}

// Init initializes the right pane
func (r *RightPane) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (r *RightPane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "left", "h":
			if r.activeTab > TabMeta {
				r.activeTab--
			}
		case "right", "l":
			if r.activeTab < TabResponse {
				r.activeTab++
			}
		case "1":
			r.activeTab = TabMeta
		case "2":
			r.activeTab = TabRequest
		case "3":
			r.activeTab = TabResponse
		}
	}
	return r, nil
}

// View renders the right pane
func (r *RightPane) View() string {
	if r.request == nil {
		return r.renderEmptyState()
	}

	var content strings.Builder

	content.WriteString(r.renderTabBar())
	content.WriteString("\n\n")
	switch r.activeTab {
	case TabMeta:
		content.WriteString(r.renderMetaTab())
	case TabRequest:
		content.WriteString(r.renderRequestTab())
	case TabResponse:
		content.WriteString(r.renderResponseTab())
	}

	return content.String()
}

// renderTabBar renders the tab bar
func (r *RightPane) renderTabBar() string {
	tabs := []string{"Meta", "Request", "Response"}
	tabColors := []string{"#5D95FF", "#2E77FF", "#5D95FF"}

	var tabStrings []string
	for i, tab := range tabs {
		style := lipgloss.NewStyle().
			Padding(0, 1).
			MarginRight(1)

		if int(r.activeTab) == i {
			style = style.
				Background(lipgloss.Color(tabColors[i])).
				Foreground(lipgloss.Color("white")).
				Bold(true)
		} else {
			style = style.
				Foreground(lipgloss.Color(tabColors[i]))
		}

		tabStrings = append(tabStrings, style.Render(tab))
	}

	return lipgloss.JoinHorizontal(lipgloss.Left, tabStrings...)
}

// renderEmptyState renders the empty state
func (r *RightPane) renderEmptyState() string {
	emptyScreen := components.NewEmptyScreenComponent(
		"No Request Selected",
		"Select a request from the left pane to view its details.",
	).SetAction("Use ↑/↓ to navigate and Enter to select")

	return emptyScreen.Render()
}

// renderMetaTab renders the meta information tab
func (r *RightPane) renderMetaTab() string {
	var content strings.Builder

	content.WriteString(lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#5D95FF")).
		Render("Request Information"))
	content.WriteString("\n\n")

	info := map[string]string{
		"ID":          r.request.ID,
		"Method":      r.request.Method,
		"URL":         r.request.URL,
		"Path":        r.request.Path,
		"Status Code": fmt.Sprintf("%d", r.request.StatusCode),
		"Duration":    r.request.FormatDuration(),
		"Timestamp":   r.request.Timestamp.Format(time.RFC3339),
		"Client IP":   r.request.ClientIP,
		"User Agent":  r.request.UserAgent,
	}

	for key, value := range info {
		keyStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#5D95FF")).
			Width(12)
		valueStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("white"))

		content.WriteString(fmt.Sprintf("%s: %s\n",
			keyStyle.Render(key),
			valueStyle.Render(value)))
	}

	if r.request.Response != nil {
		content.WriteString("\n")
		content.WriteString(lipgloss.NewStyle().Bold(true).Render("Response Information"))
		content.WriteString("\n\n")

		respInfo := map[string]string{
			"Status Code": fmt.Sprintf("%d", r.request.Response.StatusCode),
			"Size":        fmt.Sprintf("%d bytes", r.request.Response.Size),
			"Duration":    r.request.Response.Duration.String(),
		}

		for key, value := range respInfo {
			keyStyle := lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("green")).
				Width(12)
			valueStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("white"))

			content.WriteString(fmt.Sprintf("%s: %s\n",
				keyStyle.Render(key),
				valueStyle.Render(value)))
		}
	}

	return content.String()
}

// renderRequestTab renders the request details tab
func (r *RightPane) renderRequestTab() string {
	var content strings.Builder

	content.WriteString(lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#2E77FF")).
		Render("Request Headers"))
	content.WriteString("\n\n")

	for key, value := range r.request.Headers {
		keyStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#2E77FF"))
		valueStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("white"))

		content.WriteString(fmt.Sprintf("%s: %s\n",
			keyStyle.Render(key),
			valueStyle.Render(value)))
	}

	if r.request.Body != "" {
		content.WriteString("\n")
		content.WriteString(lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#2E77FF")).
			Render("Request Body"))
		content.WriteString("\n\n")

		bodyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("white")).
			Background(lipgloss.Color("black")).
			Padding(1).
			Border(lipgloss.RoundedBorder())

		body := r.request.Body
		if len(body) > 500 {
			body = body[:500] + "\n... (truncated)"
		}

		content.WriteString(bodyStyle.Render(body))
	}

	return content.String()
}

// renderResponseTab renders the response details tab
func (r *RightPane) renderResponseTab() string {
	var content strings.Builder

	if r.request.Response == nil {
		content.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("gray")).
			Render("No response data available"))
		return content.String()
	}

	content.WriteString(lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#5D95FF")).
		Render("Response Headers"))
	content.WriteString("\n\n")

	for key, value := range r.request.Response.Headers {
		keyStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#5D95FF"))
		valueStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("white"))

		content.WriteString(fmt.Sprintf("%s: %s\n",
			keyStyle.Render(key),
			valueStyle.Render(value)))
	}

	if r.request.Response.Body != "" {
		content.WriteString("\n")
		content.WriteString(lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#5D95FF")).
			Render("Response Body"))
		content.WriteString("\n\n")

		bodyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("white")).
			Background(lipgloss.Color("black")).
			Padding(1).
			Border(lipgloss.RoundedBorder())

		body := r.request.Response.Body
		if len(body) > 500 {
			body = body[:500] + "\n... (truncated)"
		}

		content.WriteString(bodyStyle.Render(body))
	}

	return content.String()
}
