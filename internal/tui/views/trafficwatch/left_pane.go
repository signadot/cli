package trafficwatch

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/bubbles/paginator"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/signadot/libconnect/common/trafficwatch/api"
)

type LeftPane struct {
	requests []api.RequestMetadata
	selected int
	width    int
	height   int

	paginator paginator.Model
}

type RefreshDataMsg struct {
	Requests []api.RequestMetadata
}

func NewLeftPane(requests []api.RequestMetadata) *LeftPane {

	p := paginator.New()
	p.Type = paginator.Arabic
	p.ArabicFormat = "%d of %d"

	return &LeftPane{
		requests:  requests,
		selected:  -1, // No element selected by default
		width:     50,
		height:    20,
		paginator: p,
	}
}

func (l *LeftPane) SetSize(width, height int) {
	l.width = width
	l.height = height

	if len(l.requests) != 0 {
		itemHeight := lipgloss.Height(l.renderRequestItem(l.requests[0], true)) // Using true to have in calculation the selected item
		l.paginator.PerPage = height / (itemHeight)
	}

	// Elements per page is the available height divided by the height of a single item

	// Calculate the total number of pages, making sure to round up
	l.paginator.TotalPages = int(math.Ceil(float64(len(l.requests)) / float64(l.paginator.PerPage)))

	// Calculate the page based on the selected index
	l.paginator.Page = l.selected / l.paginator.PerPage

	if l.paginator.Page >= l.paginator.TotalPages {
		l.paginator.Page = l.paginator.TotalPages - 1
	}

	if l.paginator.Page < 0 {
		l.paginator.Page = 0
	}
}

func (l *LeftPane) SetRequests(requests []api.RequestMetadata) {
	l.requests = requests
	if l.selected >= len(requests) && l.selected != -1 {
		l.selected = 0
	}
}

type NextPageMsg struct {
	Page int

	AutoFirst bool // When true, the selected index will not be changed
}

type PrevPageMsg struct {
	Page int

	AutoLast bool // When true, the selected index will not be changed
}

func (l *LeftPane) NextPage(withAuto bool) tea.Cmd {
	return func() tea.Msg {
		return NextPageMsg{
			Page:      l.paginator.Page + 1,
			AutoFirst: withAuto,
		}
	}
}

func (l *LeftPane) PrevPage(withAuto bool) tea.Cmd {
	return func() tea.Msg {
		return PrevPageMsg{
			Page:     l.paginator.Page - 1,
			AutoLast: withAuto,
		}
	}
}

func (l *LeftPane) RefreshData(requests []api.RequestMetadata) tea.Cmd {
	return func() tea.Msg {
		return RefreshDataMsg{Requests: requests}
	}
}
func (l *LeftPane) Init() tea.Cmd {
	return nil
}

func (l *LeftPane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case RefreshDataMsg:
		l.SetRequests(msg.Requests)
		l.SetSize(l.width, l.height)
		return l, nil
	case NextPageMsg:
		if l.paginator.Page < l.paginator.TotalPages {
			l.paginator.Page++

			if (l.paginator.Page) >= l.paginator.TotalPages {
				l.paginator.Page--
				return l, nil
			}

			// If auto first, keep the selected index as is
			// If not auto first, move the selected index down by the number of items per page
			if !msg.AutoFirst {
				l.selected = l.selected + l.paginator.PerPage
			}
		}
		return l, nil
	case PrevPageMsg:
		if l.paginator.Page > 0 {

			// If not auto last, move the selected index up by the number of items per page
			// If auto last, keep the selected index as is
			if !msg.AutoLast {
				l.selected = l.selected - l.paginator.PerPage
			}

			l.paginator.Page--
		}
		return l, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "up":
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
		case "down":
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
		case "right":
			// Right arrow should move focus to right pane
			// This will be handled by the main view
			return l, nil
		}
	}

	var cmd tea.Cmd
	l.paginator, cmd = l.paginator.Update(msg)
	return l, cmd
}

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

	start, end = l.paginator.GetSliceBounds(len(l.requests))

	for i := start; i < end; i++ {
		req := l.requests[i]
		item := l.renderRequestItem(req, i == l.selected)
		content.WriteString(item)
		if i < end-1 { // Don't add newline after the last item
			content.WriteString("\n")
		}
	}

	content.WriteString("\n" + l.paginator.View())

	return content.String()
}

// renderRequestItem renders a single request item
func (l *LeftPane) renderRequestItem(req api.RequestMetadata, selected bool) string {
	methodColor := lipgloss.Color("blue")
	methodStyle := lipgloss.NewStyle().
		Foreground(methodColor).
		Bold(true).
		Width(6)
	method := methodStyle.Render(req.Method)

	statusColor := lipgloss.Color("green")
	statusStyle := lipgloss.NewStyle().
		Foreground(statusColor).
		Bold(true).
		Width(4)
	status := statusStyle.Render(fmt.Sprintf("%d", 200))

	// Show request URI instead of path
	requestURI := req.RequestURI
	if len(requestURI) > l.width-15 {
		requestURI = requestURI[:l.width-18] + "..."
	}

	// Show routing key
	routingKey := req.MiddlewareRequestID
	if len(routingKey) > 20 {
		routingKey = routingKey[:17] + "..."
	}

	duration := "100ms"

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

func (l *LeftPane) renderEmptyState() string {
	return lipgloss.NewStyle().
		Align(lipgloss.Center).
		Foreground(lipgloss.Color("gray")).
		Render("No traffic data available")
}

func (l *LeftPane) sendSelection() tea.Cmd {

	mixIndex := l.paginator.PerPage * l.paginator.Page
	maxIndex := mixIndex + l.paginator.PerPage - 1

	if l.selected < len(l.requests) {

		var cmd tea.Cmd
		if l.selected < mixIndex {
			cmd = l.PrevPage(true)
		} else if l.selected > maxIndex {
			cmd = l.NextPage(true)
		}

		return tea.Batch(cmd, func() tea.Msg {
			return RequestSelectedMsg{
				RequestID: l.requests[l.selected].MiddlewareRequestID,
			}
		})

	}
	return nil
}
