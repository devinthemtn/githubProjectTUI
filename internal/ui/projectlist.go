package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/thomaskoefod/githubProjectTUI/internal/models"
)

var (
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("#7D56F4"))
)

// projectItem implements list.Item for the project list
type projectItem struct {
	project models.Project
}

func (i projectItem) FilterValue() string { return i.project.Title }
func (i projectItem) Title() string       { return i.project.Title }
func (i projectItem) Description() string {
	desc := i.project.ShortDescription
	if desc == "" {
		desc = "No description"
	}
	status := "Open"
	if i.project.Closed {
		status = "Closed"
	}
	visibility := "Private"
	if i.project.Public {
		visibility = "Public"
	}
	return fmt.Sprintf("%s • %s • %d items", status, visibility, i.project.ItemCount)
}

// ProjectListModel represents the project list view
type ProjectListModel struct {
	list     list.Model
	projects []models.Project
	width    int
	height   int
}

func NewProjectListModel(projects []models.Project) ProjectListModel {
	items := make([]list.Item, len(projects))
	for i, p := range projects {
		items[i] = projectItem{project: p}
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "GitHub Projects"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)  // Use custom help in footer
	l.Styles.Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4")).
		MarginLeft(2)

	return ProjectListModel{
		list:     l,
		projects: projects,
	}
}

func (m ProjectListModel) Init() tea.Cmd {
	return nil
}

func (m ProjectListModel) Update(msg tea.Msg) (ProjectListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetWidth(msg.Width)
		// Use full height minus small margin for status/help
		m.list.SetHeight(msg.Height - 2)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			// Handle selection
			if i, ok := m.list.SelectedItem().(projectItem); ok {
				return m, SelectProjectCmd(i.project)
			}
		case "n":
			// Create new project
			return m, NewProjectCmd()
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m ProjectListModel) View() string {
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#626262")).
		Padding(0, 2)

	help := helpStyle.Render("enter: open • n: new project • esc: back • /: filter • q: quit")
	
	return lipgloss.JoinVertical(lipgloss.Left, m.list.View(), help)
}

func (m ProjectListModel) GetSelectedProject() *models.Project {
	if i, ok := m.list.SelectedItem().(projectItem); ok {
		return &i.project
	}
	return nil
}

// SelectProjectCmd is a command to signal project selection
func SelectProjectCmd(project models.Project) tea.Cmd {
	return func() tea.Msg {
		return ProjectSelectedMsg{Project: project}
	}
}

// NewProjectCmd signals creating a new project
func NewProjectCmd() tea.Cmd {
	return func() tea.Msg {
		return NewProjectMsg{}
	}
}

// ProjectSelectedMsg is sent when a project is selected
type ProjectSelectedMsg struct {
	Project models.Project
}

// NewProjectMsg is sent when user wants to create a new project
type NewProjectMsg struct{}

