package trafficwatch

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/signadot/cli/internal/tui/components"
	"github.com/signadot/cli/internal/tui/filemanager"
	"github.com/signadot/cli/internal/tui/utils"
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
	TabRequest RightPaneTab = iota
	TabResponse
)

// RightPane represents the right pane showing request details
type RightPane struct {
	recordDir string
	request   *filemanager.RequestMetadata
	activeTab RightPaneTab
	width     int
	height    int

	focused bool

	requestContent  string
	responseContent string

	viewport viewport.Model
}

// NewRightPane creates a new right pane
func NewRightPane(recordDir string) *RightPane {
	return &RightPane{
		recordDir: recordDir,
		activeTab: TabRequest,
		width:     50,
		height:    20,
		viewport:  viewport.New(40, 20),
	}
}

func (r *RightPane) SetFocused(focused bool) {
	r.focused = focused
}

// SetSize sets the size of the right pane
func (r *RightPane) SetSize(width, height int) {
	r.width = width
	r.height = height

	r.viewport.Height = r.height - lipgloss.Height(r.renderTabBar(false)) - 4
	r.viewport.Width = width - 1
	r.viewport.YPosition = lipgloss.Height(r.renderTabBar(false))

	if r.request != nil {
		r.SetRequest(r.request)
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
			if r.activeTab > TabRequest {
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
			r.activeTab = TabRequest
		case "2":
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

	tabBar := r.renderTabBar(r.focused)
	content.WriteString(tabBar)
	content.WriteString("\n\n")

	r.viewport.SetContent(r.view())
	r.viewport.YPosition = lipgloss.Height(tabBar)

	content.WriteString(r.viewport.View())
	return content.String()
}

// renderTabBar renders the tab bar
func (r *RightPane) renderTabBar(isFocused bool) string {
	tabs := []string{"Request", "Response"}
	tabColors := []string{"#008080", "#008080"}

	var tabStrings []string
	for i, tab := range tabs {
		style := lipgloss.NewStyle().
			Padding(0, 1).
			MarginRight(1)

		if int(r.activeTab) == i {
			style = style.
				Foreground(lipgloss.Color(tabColors[i])).
				Bold(true)

			switch isFocused {
			case true:
				style = style.
					Background(lipgloss.Color(tabColors[i])).
					Foreground(lipgloss.Color("white"))
			case false:
				style = style.
					Underline(true)
			}
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

// renderRequestTab renders the request details tab
func (r *RightPane) renderRequestTab(reqMeta *filemanager.RequestMetadata, req *http.Request, err error) string {
	var content strings.Builder

	if err != nil {
		content.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("red")).
			Render(err.Error()))
		return content.String()
	}

	// render general section
	content.WriteString(lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#008080")).
		Render("General"))
	content.WriteString("\n\n")
	content.WriteString(r.getLineRenderMeta("ID", reqMeta.MiddlewareRequestID))
	content.WriteString(r.getLineRenderMeta("URL", reqMeta.RequestURI))
	content.WriteString(r.getLineRenderMeta("Protocol", req.Proto))
	content.WriteString(r.getLineRenderMeta("Method", req.Method))
	content.WriteString(r.getLineRenderMeta("Routing Key", reqMeta.RoutingKey))
	content.WriteString(r.getLineRenderMeta("Workload", reqMeta.DestWorkload))
	content.WriteString(r.getLineRenderMeta("File",
		filemanager.GetSourceRequestPath(r.recordDir, reqMeta.MiddlewareRequestID)))
	content.WriteString("\n")

	// render headers section
	content.WriteString(lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#008080")).
		Render("Headers"))
	content.WriteString("\n\n")

	for key, values := range req.Header {
		content.WriteString(r.getLineRenderMeta(key, strings.Join(values, ", ")))
	}

	// render body section
	r.renderBody(&content, req, nil)

	return content.String()
}

// renderResponseTab renders the response details tab
func (r *RightPane) renderResponseTab(reqMeta *filemanager.RequestMetadata, resp *http.Response, err error) string {
	var content strings.Builder

	if err != nil {
		content.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("red")).
			Render(err.Error()))
		return content.String()
	}

	if resp.StatusCode == 0 {
		content.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("gray")).
			Render("No response data available"))
		return content.String()
	}

	// render general section
	content.WriteString(lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#008080")).
		Render("General"))
	content.WriteString("\n\n")
	content.WriteString(r.getLineRenderMeta("Status", resp.Status))
	content.WriteString(r.getLineRenderMeta("Protocol", resp.Proto))
	content.WriteString(r.getLineRenderMeta("File",
		filemanager.GetSourceResponsePath(r.recordDir, reqMeta.MiddlewareRequestID)))
	content.WriteString("\n")

	// render headers section
	content.WriteString(lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#008080")).
		Render("Headers"))
	content.WriteString("\n\n")

	for key, values := range resp.Header {
		content.WriteString(r.getLineRenderMeta(key, strings.Join(values, ", ")))
	}

	// render body section
	r.renderBody(&content, nil, resp)

	return content.String()
}

func (r *RightPane) renderBody(content *strings.Builder, req *http.Request, resp *http.Response) {
	var (
		body   io.ReadCloser
		length int64
	)
	if req != nil {
		body = req.Body
		length = req.ContentLength
	} else if resp != nil {
		body = resp.Body
		length = resp.ContentLength
	}
	if body == nil {
		return
	}

	content.WriteString("\n")
	content.WriteString(lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#008080")).
		Render("Body"))
	content.WriteString("\n\n")

	var isRenderable bool
	if req != nil {
		isRenderable = isReqBodyRenderable(req)
	} else {
		isRenderable = isRespBodyRenderable(resp)
	}
	if !isRenderable {
		content.WriteString("Unsupported content type for terminal rendering")
		return
	}
	if length > 500*1024 /*500KB*/ {
		content.WriteString("Body too large to display in the terminal")
		return
	}

	// try reading the body
	var bodyString string
	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		bodyString = fmt.Sprintf("Failed to read body: %v", err.Error())
	} else {
		bodyString = string(bodyBytes)
	}
	if len(bodyString) == 0 {
		content.WriteString("No body content available")
		return
	}

	bodyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("white")).
		Background(lipgloss.Color("black")).
		Padding(1).
		Width(r.width - 6).
		Border(lipgloss.RoundedBorder())

	content.WriteString(bodyStyle.Render(string(bodyString)))
}

// SetRequest sets the current request to display
func (r *RightPane) SetRequest(reqMeta *filemanager.RequestMetadata) {
	r.request = reqMeta

	// Load the request/response details from the os
	requestDetail, err := utils.LoadHttpRequest(
		filemanager.GetSourceRequestPath(r.recordDir, reqMeta.MiddlewareRequestID))
	r.requestContent = r.renderRequestTab(reqMeta, requestDetail, err)

	responseDetail, err := utils.LoadHttpResponse(
		filemanager.GetSourceResponsePath(r.recordDir, reqMeta.MiddlewareRequestID))
	r.responseContent = r.renderResponseTab(reqMeta, responseDetail, err)
}
