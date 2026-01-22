package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/thomaskoefod/githubProjectTUI/internal/models"
)

// repositoryItem represents a repository in the list
type repositoryItem struct {
	repo models.Repository
}

func (i repositoryItem) FilterValue() string { return i.repo.Name }
func (i repositoryItem) Title() string {
	visibility := "🔒"
	if !i.repo.IsPrivate {
		visibility = "🔓"
	}
	return fmt.Sprintf("%s %s/%s", visibility, i.repo.Owner, i.repo.Name)
}
func (i repositoryItem) Description() string {
	if i.repo.Description != "" {
		return i.repo.Description
	}
	return "No description"
}

// RepositorySelectorModel represents the repository selection view
type RepositorySelectorModel struct {
	list    list.Model
	repos   []models.Repository
	project models.Project
	item    models.ProjectItem
	width   int
	height  int
}

func NewRepositorySelectorModel(repos []models.Repository, project models.Project, item models.ProjectItem) RepositorySelectorModel {
	items := make([]list.Item, len(repos))
	for i, repo := range repos {
		items[i] = repositoryItem{repo: repo}
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = ""
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true) // Enable filtering for searching
	l.SetShowHelp(false)
	l.Styles.Title = lipgloss.NewStyle()

	return RepositorySelectorModel{
		list:    l,
		repos:   repos,
		project: project,
		item:    item,
	}
}

func (m RepositorySelectorModel) Init() tea.Cmd {
	return nil
}

func (m RepositorySelectorModel) Update(msg tea.Msg) (RepositorySelectorModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetWidth(msg.Width)
		m.list.SetHeight(msg.Height - 6)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if i, ok := m.list.SelectedItem().(repositoryItem); ok {
				return m, ConvertDraftCmd(m.project, m.item, i.repo)
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m RepositorySelectorModel) View() string {
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#626262")).
		Padding(1, 2)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4")).
		Padding(1, 2)

	help := helpStyle.Render("↑/↓: navigate • /: search • enter: select • esc: cancel")
	header := titleStyle.Render(fmt.Sprintf("Convert \"%s\" to issue - Select repository:", m.item.Title))

	return lipgloss.JoinVertical(lipgloss.Left, header, m.list.View(), help)
}

// ConvertDraftCmd signals draft conversion
func ConvertDraftCmd(project models.Project, item models.ProjectItem, repo models.Repository) tea.Cmd {
	return func() tea.Msg {
		return ConvertDraftMsg{
			Project:    project,
			Item:       item,
			Repository: repo,
		}
	}
}

// ConvertDraftMsg is sent when converting a draft to an issue
type ConvertDraftMsg struct {
	Project    models.Project
	Item       models.ProjectItem
	Repository models.Repository
}

// RepositoriesLoadedMsg is sent when repositories are loaded
type RepositoriesLoadedMsg struct {
	Repositories []models.Repository
	Project      models.Project
	Item         models.ProjectItem
}

// DraftConvertedMsg is sent when a draft is successfully converted
type DraftConvertedMsg struct {
	Project models.Project
}
