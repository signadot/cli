package trafficwatch

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/signadot/cli/internal/tui/components"
	"github.com/signadot/cli/internal/tui/utils"
	"github.com/signadot/libconnect/common/trafficwatch/api"
)

var (
	keyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#5D95FF")).
			Width(18)

	valueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("white"))
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
	request   *api.RequestMetadata
	activeTab RightPaneTab
	width     int
	height    int

	currentTrafficDir string
	metadataContent   string
	requestContent    string
	responseContent   string

	viewport viewport.Model
}

// NewRightPane creates a new right pane
func NewRightPane() *RightPane {
	return &RightPane{
		activeTab: TabMeta,
		width:     50,
		height:    20,
		viewport:  viewport.New(40, 20),
	}
}

// SetSize sets the size of the right pane
func (r *RightPane) SetSize(width, height int) {
	r.width = width
	r.height = height

	r.viewport.Height = r.height - lipgloss.Height(r.renderTabBar()) - 4
	r.viewport.Width = width - 1
	r.viewport.YPosition = lipgloss.Height(r.renderTabBar())

	if r.request != nil {
		r.SetRequest(r.currentTrafficDir, r.request)
	}
}

// GetActiveTab returns the currently active tab
func (r *RightPane) GetActiveTab() RightPaneTab {
	return r.activeTab
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
		case "left":
			if r.activeTab > TabMeta {
				r.activeTab--
			} else {
				// If at first tab, left arrow should move focus back to left pane
				// This will be handled by the main view
			}
		case "right":
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

	var cmd tea.Cmd
	r.viewport, cmd = r.viewport.Update(msg)

	return r, cmd
}

// View renders the right pane
func (r *RightPane) view() string {

	var content strings.Builder

	switch r.activeTab {
	case TabMeta:
		content.WriteString(r.metadataContent)
	case TabRequest:
		content.WriteString(r.requestContent)
	case TabResponse:
		content.WriteString(r.responseContent)
	}

	return content.String()
}

func (r *RightPane) View() string {
	if r.request == nil {
		return r.renderEmptyState()
	}

	var content strings.Builder

	tabBar := r.renderTabBar()
	content.WriteString(tabBar)
	content.WriteString("\n\n")

	r.viewport.SetContent(r.view())
	r.viewport.YPosition = lipgloss.Height(tabBar)

	content.WriteString(r.viewport.View())
	return content.String()
}

// renderTabBar renders the tab bar
func (r *RightPane) renderTabBar() string {
	tabs := []string{"Meta", "Request", "Response"}
	tabColors := []string{"#2E77FF", "#2E77FF", "#2E77FF"}

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
		r.width,
		r.height,
	).SetAction("Use ↑/↓ to navigate and Enter to select")

	return emptyScreen.Render()
}

func (r *RightPane) getLineRenderMeta(key string, value string) string {
	if value == "" {
		return ""
	}

	content := fmt.Sprintf("%s: %s\n", keyStyle.Render(key), valueStyle.Render(value))
	width := lipgloss.Width(content)
	if width > r.width { // If the content is too long, split it into multiple lines
		v := valueStyle.SetString(value).Width(r.width - 4).PaddingLeft(2).Render()

		return fmt.Sprintf("%s⤶\n%s\n", keyStyle.SetString(key), v)
	}

	return content
}

// renderMetaTab renders the meta information tab
func (r *RightPane) renderMetaTab(request *api.RequestMetadata) string {
	var content strings.Builder

	content.WriteString(lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#5D95FF")).
		Render("Request Information"))
	content.WriteString("\n\n")

	content.WriteString(r.getLineRenderMeta("Middleware Request", request.MiddlewareRequestID))
	content.WriteString(r.getLineRenderMeta("Routing Key", request.RoutingKey))
	content.WriteString(r.getLineRenderMeta("Method", request.Method))
	content.WriteString(r.getLineRenderMeta("Request URI", request.RequestURI))
	content.WriteString(r.getLineRenderMeta("Host", request.Host))
	content.WriteString(r.getLineRenderMeta("Dest Workload", request.DestWorkload))
	content.WriteString(r.getLineRenderMeta("Protocol", request.Proto))
	content.WriteString(r.getLineRenderMeta("User Agent", request.UserAgent))

	return content.String()
}

// renderRequestTab renders the request details tab
func (r *RightPane) renderRequestTab(request *http.Request) string {
	var content strings.Builder

	content.WriteString(lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#2E77FF")).
		Render("Request Headers"))
	content.WriteString("\n\n")

	for key, values := range request.Header {
		content.WriteString(r.getLineRenderMeta(key, strings.Join(values, ", ")))
	}

	if request.Body != nil {
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

		bodyString, err := io.ReadAll(request.Body)
		if err != nil {
			log.Fatal(err)
		}

		if len(bodyString) == 0 {
			content.WriteString("No body data available")
			return content.String()
		}

		content.WriteString(bodyStyle.Render(string(bodyString)))
	}

	return content.String()
}

// renderResponseTab renders the response details tab
func (r *RightPane) renderResponseTab(response *http.Response) string {
	var content strings.Builder

	if response.StatusCode == 0 {
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

	for key, values := range response.Header {
		content.WriteString(r.getLineRenderMeta(key, strings.Join(values, ", ")))
	}

	if response.Body != nil {
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
			Width(r.width - 6).
			Border(lipgloss.RoundedBorder())

		body := response.Body
		bodyString, err := io.ReadAll(body)
		if err != nil {
			log.Fatal(err)
		}

		if len(bodyString) == 0 {
			content.WriteString("No body data available")
			return content.String()
		}

		content.WriteString(bodyStyle.Render(string(bodyString)))
	}

	return content.String()
}

// SetRequest sets the current request to display
func (r *RightPane) SetRequest(trafficDir string, request *api.RequestMetadata) {
	r.request = request

	r.currentTrafficDir = trafficDir

	// Load the request detail from the /traffic-dir/request-id
	requestDetail, err := utils.LoadHttpRequest(filepath.Join(trafficDir, request.MiddlewareRequestID, "request"))
	if err != nil {
		log.Fatal(err)
	}
	responseDetail, err := utils.LoadHttpResponse(filepath.Join(trafficDir, request.MiddlewareRequestID, "response"))
	if err != nil {
		log.Fatal(err)
	}

	r.metadataContent = r.renderMetaTab(request)
	r.requestContent = r.renderRequestTab(requestDetail)
	r.responseContent = r.renderResponseTab(responseDetail)
}
