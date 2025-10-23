package trafficwatch

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/signadot/cli/internal/tui/models"
)

// LeftPane represents the left pane showing HTTP requests
type LeftPane struct {
	requests []models.HTTPRequest
	selected int
	width    int
	height   int
}

// NewLeftPane creates a new left pane
func NewLeftPane(requests []models.HTTPRequest) *LeftPane {
	return &LeftPane{
		requests: requests,
		selected: -1, // No element selected by default
		width:    50,
		height:   20,
	}
}

// SetSize sets the size of the left pane
func (l *LeftPane) SetSize(width, height int) {
	l.width = width
	l.height = height
}

// SetRequests updates the requests list
func (l *LeftPane) SetRequests(requests []models.HTTPRequest) {
	l.requests = requests
	if l.selected >= len(requests) && l.selected != -1 {
		l.selected = 0
	}
}

// Init initializes the left pane
func (l *LeftPane) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (l *LeftPane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if l.selected > 0 {
				// If an item is selected, move up
				l.selected--
				return l, l.sendSelection()
			} else if l.selected == -1 && len(l.requests) > 0 {
				// If nothing is selected, go to the last item
				l.selected = len(l.requests) - 1
				return l, l.sendSelection()
			}
			return l, nil
		case "down", "j":
			if l.selected < len(l.requests)-1 {
				// If an item is selected, move down
				l.selected++
				return l, l.sendSelection()
			} else if l.selected == -1 && len(l.requests) > 0 {
				// If nothing is selected, go to the first item
				l.selected = 0
				return l, l.sendSelection()
			}
			return l, nil
		case "enter":
			return l, l.sendSelection()
		case "right":
			// Right arrow should move focus to right pane
			// This will be handled by the main view
			return l, nil
		}
	}
	return l, nil
}

// View renders the left pane
func (l *LeftPane) View() string {
	if len(l.requests) == 0 {
		return l.renderEmptyState()
	}

	var content strings.Builder

	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#5D95FF")).
		Render(fmt.Sprintf("Traffic Watch (%d)", len(l.requests)))
	content.WriteString(header)
	content.WriteString("\n\n")
	start := 0
	end := len(l.requests)

	// Account for two lines per request (height-3)/2
	maxVisibleRequests := (l.height - 3) / 2
	if len(l.requests) > maxVisibleRequests {
		if l.selected > maxVisibleRequests {
			start = l.selected - maxVisibleRequests
		}
		end = start + maxVisibleRequests
		if end > len(l.requests) {
			end = len(l.requests)
		}
	}

	for i := start; i < end; i++ {
		req := l.requests[i]
		item := l.renderRequestItem(req, i == l.selected)
		content.WriteString(item)
		if i < end-1 { // Don't add newline after the last item
			content.WriteString("\n")
		}
	}

	return content.String()
}

// renderRequestItem renders a single request item
func (l *LeftPane) renderRequestItem(req models.HTTPRequest, selected bool) string {
	methodColor := lipgloss.Color(req.GetMethodColor())
	methodStyle := lipgloss.NewStyle().
		Foreground(methodColor).
		Bold(true).
		Width(6)
	method := methodStyle.Render(req.Method)

	statusColor := lipgloss.Color(req.GetStatusColor())
	statusStyle := lipgloss.NewStyle().
		Foreground(statusColor).
		Bold(true).
		Width(4)
	status := statusStyle.Render(fmt.Sprintf("%d", req.StatusCode))

	// Show request URI instead of path
	requestURI := req.RequestURI
	if len(requestURI) > l.width-15 {
		requestURI = requestURI[:l.width-18] + "..."
	}

	// Show routing key
	routingKey := req.RoutingKey
	if len(routingKey) > 20 {
		routingKey = routingKey[:17] + "..."
	}

	duration := req.FormatDuration()

	indicator := "  "
	if selected {
		indicator = "> "
	}

	// First line: method status duration
	line1 := fmt.Sprintf("%s%s %s %s", indicator, method, status, duration)
	
	// Second line: requestURI [routingKey]
	line2 := fmt.Sprintf("  %s [%s]", requestURI, routingKey)

	// Combine both lines
	lines := []string{line1, line2}
	
	if selected {
		line1Style := lipgloss.NewStyle().
			Background(lipgloss.Color("#2E77FF")).
			Foreground(lipgloss.Color("white")).
			Bold(true).
			Padding(0, 1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#2E77FF"))
		
		line2Style := lipgloss.NewStyle().
			Background(lipgloss.Color("#2E77FF")).
			Foreground(lipgloss.Color("white")).
			Padding(0, 1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#2E77FF"))
		
		lines[0] = line1Style.Render(line1)
		lines[1] = line2Style.Render(line2)
	}

	return strings.Join(lines, "\n")
}

// renderEmptyState renders the empty state
func (l *LeftPane) renderEmptyState() string {
	return lipgloss.NewStyle().
		Align(lipgloss.Center).
		Foreground(lipgloss.Color("gray")).
		Render("No traffic data available")
}

// sendSelection sends a selection message
func (l *LeftPane) sendSelection() tea.Cmd {
	if l.selected < len(l.requests) {
		return func() tea.Msg {
			return RequestSelectedMsg{
				RequestID: l.requests[l.selected].ID,
			}
		}
	}
	return nil
}
