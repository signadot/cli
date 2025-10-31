package trafficwatch

import (
	"fmt"
	"math"
	"net/url"
	"strings"

	"github.com/charmbracelet/bubbles/paginator"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/signadot/cli/internal/tui/filemanager"
)

type LeftPane struct {
	requests   []*filemanager.RequestMetadata
	selected   int
	followMode bool

	width  int
	height int

	paginator paginator.Model
}

func NewLeftPane(requests []*filemanager.RequestMetadata) *LeftPane {
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
		availableHeight := height - 6
		itemHeight := lipgloss.Height(l.renderRequestItem(l.requests[0], true)) // Using true to have in calculation the selected item
		l.paginator.PerPage = availableHeight / itemHeight                      // Elements per page is the available height divided by the height of a single item
	}

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

func (l *LeftPane) SetRequests(requests []*filemanager.RequestMetadata) {
	l.requests = requests
	if l.selected >= len(requests) && l.selected != -1 {
		l.selected = 0
	}
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

func (l *LeftPane) RefreshData(requests []*filemanager.RequestMetadata) tea.Cmd {
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

	case ToggleFollowModeMsg:
		l.followMode = !l.followMode
		return l, nil
	case NextPageMsg:
		l.unsetFollowMode()
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

				// This happens when the same row in the next page, don't have a pair element in that row
				if l.selected >= len(l.requests) {
					l.selected = len(l.requests) - 1
				}
				return l, l.sendSelection()
			}
		}
		return l, nil
	case PrevPageMsg:
		l.unsetFollowMode()

		if l.paginator.Page > 0 {

			// If not auto last, move the selected index up by the number of items per page
			// If auto last, keep the selected index as is
			if !msg.AutoLast {
				l.selected = l.selected - l.paginator.PerPage
				return l, l.sendSelection()
			}

			l.paginator.Page--
		}
		return l, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "up":
			l.unsetFollowMode()

			if l.selected > 0 {
				// If an item is selected, move up
				l.selected--
				return l, l.sendSelection()
			}
			return l, nil
		case "down":
			l.unsetFollowMode()

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

	start, end := l.paginator.GetSliceBounds(len(l.requests))
	for i := start; i < end; i++ {
		req := l.requests[i]
		item := l.renderRequestItem(req, i == l.selected)
		content.WriteString(item)
		if i < end-1 { // Don't add newline after the last item
			content.WriteString("\n")
		}
	}

	mainContent := content.String()
	paginatorView := l.paginator.View()

	// Use lipgloss to position content at top and paginator at bottom
	paginatorHeight := lipgloss.Height(paginatorView)
	availableHeight := l.height - paginatorHeight - 2

	// Place main content at the top, paginator at the bottom
	topContent := lipgloss.NewStyle().
		Height(availableHeight).
		Render(mainContent)

	return lipgloss.JoinVertical(lipgloss.Left, topContent, paginatorView)
}

func (l *LeftPane) renderRequestItem(req *filemanager.RequestMetadata, selected bool) string {
	parsedURL, err := url.ParseRequestURI(req.RequestURI)
	if err != nil {
		parsedURL = &url.URL{Path: req.RequestURI}
	}

	cMethodGet := lipgloss.Color("10")      // bright green
	cMethodPost := lipgloss.Color("14")     // cyan
	cMethodPut := lipgloss.Color("214")     // orange
	cMethodDel := lipgloss.Color("9")       // red
	cHost := lipgloss.Color("245")          // light gray
	cPath := lipgloss.Color("81")           // bright blue
	cSelectedAccent := lipgloss.Color("63") // blue accent (selection marker)

	var methodColor lipgloss.Color
	switch strings.ToUpper(req.Method) {
	case "GET":
		methodColor = cMethodGet
	case "POST":
		methodColor = cMethodPost
	case "PUT":
		methodColor = cMethodPut
	case "DELETE":
		methodColor = cMethodDel
	default:
		methodColor = lipgloss.Color("7") // neutral gray
	}

	methodStyle := lipgloss.NewStyle().Foreground(methodColor).Bold(true).Width(6)
	hostStyle := lipgloss.NewStyle().Foreground(cHost)
	pathStyle := lipgloss.NewStyle().Foreground(cPath)

	method := methodStyle.Render(strings.ToUpper(req.Method))
	host := hostStyle.Render(parsedURL.Host)
	fullPath := pathStyle.Render(parsedURL.Path)
	if parsedURL.RawQuery != "" {
		fullPath += "?" + parsedURL.RawQuery
	}

	if parsedURL.Fragment != "" {
		fullPath += "#" + parsedURL.Fragment
	}

	// Format timestamp (strip milliseconds and timezone)
	formattedTime := req.DoneAt.Format("2006-01-02 15:04:05")

	// Properly render protocol
	var protocol string
	switch req.Protocol {
	case filemanager.ProtocolGRPC:
		protocol = "gRPC"
	default:
		protocol = strings.ToUpper(string(req.Protocol))
	}

	// date-time  protocol  host
	line1 := fmt.Sprintf("%s  %-5s  ", formattedTime, protocol)
	line1 += truncateURL(host, l.width-lipgloss.Width(line1)-1)
	// method  fullPath
	line2 := fmt.Sprintf("%-6s  ", method)
	line2 += truncateURL(fullPath, l.width-lipgloss.Width(line2)-1)

	content := lipgloss.NewStyle().Width(l.width).Render(line1 + "\n" + line2)
	if selected {
		indicator := lipgloss.NewStyle().
			Foreground(cSelectedAccent).
			Render("▌")

		lines := strings.Split(content, "\n")
		for i, line := range lines {
			lines[i] = fmt.Sprintf("%s %s", indicator, line)
		}
		lines[len(lines)-1] += "\n"
		return strings.Join(lines, "\n")
	}

	return lipgloss.NewStyle().PaddingLeft(2).Render(content + "\n")
}

func (l *LeftPane) renderEmptyState() string {
	return lipgloss.NewStyle().
		Align(lipgloss.Center).
		Foreground(lipgloss.Color("gray")).
		Render("No traffic data available")
}

func (l *LeftPane) sendSelection() tea.Cmd {
	minIndex := l.paginator.PerPage * l.paginator.Page
	maxIndex := minIndex + l.paginator.PerPage - 1

	if l.selected < len(l.requests) {

		// If selected item is not on the current page, jump directly to the correct page
		if l.selected < minIndex || l.selected > maxIndex {
			// Calculate the page that contains the selected item
			targetPage := l.selected / l.paginator.PerPage

			// Ensure the page is within valid bounds
			if targetPage >= l.paginator.TotalPages {
				targetPage = l.paginator.TotalPages - 1
			}
			if targetPage < 0 {
				targetPage = 0
			}

			// Set the page directly
			l.paginator.Page = targetPage
		}

		return func() tea.Msg {
			return RequestSelectedMsg{
				RequestID: l.requests[l.selected].MiddlewareRequestID,
			}
		}

	}
	return nil
}

func (l *LeftPane) toggleFollowMode() {
	l.followMode = !l.followMode
}

func (l *LeftPane) unsetFollowMode() {
	l.followMode = false
}
