package trafficwatch

import (
	"os"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/signadot/cli/internal/tui/views"
)

type DirectoriesTrafficWatchViewModel struct {
	RecordDir string // The directory where the recorded traffic is stored

	commonViewModel *views.CommonViewModel

	// panes
	leftPaneViewModel  *DirectoriesPaneViewModel
	rightPaneViewModel *detailRecordPaneViewModel
}

func NewDirectoriesTrafficWatchViewModel(recordDir string) tea.Model {
	leftPaneViewModel := newDirectoriesPaneViewModel(recordDir)
	return &DirectoriesTrafficWatchViewModel{
		RecordDir:         recordDir,
		commonViewModel:   views.NewCommonViewModel(),
		leftPaneViewModel: leftPaneViewModel,
	}
}

func (m *DirectoriesTrafficWatchViewModel) Init() tea.Cmd {
	cmd := tea.Batch(m.commonViewModel.Init(), m.leftPaneViewModel.Init())
	return cmd
}

func (m *DirectoriesTrafficWatchViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var commonCmd tea.Cmd
	m.commonViewModel, commonCmd = m.commonViewModel.Update(msg)
	leftPaneViewModel, cmd := m.leftPaneViewModel.Update(msg)
	m.leftPaneViewModel = leftPaneViewModel.(*DirectoriesPaneViewModel)
	return m, tea.Batch(commonCmd, cmd)
}

func (m *DirectoriesTrafficWatchViewModel) View() string {
	commonView := m.commonViewModel.View()
	if commonView != "" {
		return commonView
	}

	leftPaneView := m.leftPaneViewModel.View()
	if leftPaneView != "" {
		return leftPaneView
	}

	return ""
}

type DirectoriesPaneViewModel struct {
	recordDir string

	// Model
	itemsDelegate list.ItemDelegate
	listModel     list.Model
}

func newDirectoriesPaneViewModel(recordDir string) *DirectoriesPaneViewModel {
	return &DirectoriesPaneViewModel{
		recordDir: recordDir,
	}
}

func (m *DirectoriesPaneViewModel) View() string {
	return m.listModel.View()
}

func (m *DirectoriesPaneViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.listModel, cmd = m.listModel.Update(msg)
	return m, cmd
}

type DirectoryItem struct {
	Title       string
	Description string
}

func (d *DirectoryItem) FilterValue() string {
	return d.Title
}

func (m *DirectoriesPaneViewModel) Init() tea.Cmd {
	m.itemsDelegate = list.NewDefaultDelegate()

	elements := []list.Item{
		&DirectoryItem{
			Title:       "All",
			Description: "All requests",
		},
	}
	entries, err := os.ReadDir(m.recordDir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				dir := entry.Name()
				elements = append(elements, &DirectoryItem{
					Title:       dir,
					Description: dir,
				})
			}
		}
	}

	m.listModel = list.New(elements, m.itemsDelegate, 0, 0)

	return nil
}

type detailRecordPaneViewModel struct {
	baseDir   string
	requestID string
}

func newDetailRecordPaneViewModel(baseDir, requestID string) *detailRecordPaneViewModel {
	return &detailRecordPaneViewModel{
		baseDir:   baseDir,
		requestID: requestID,
	}
}

func (m *detailRecordPaneViewModel) View() string {
	return ""
}

func (m *detailRecordPaneViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m *detailRecordPaneViewModel) Init() tea.Cmd {
	return nil
}
