package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/thomaskoefod/githubProjectTUI/internal/models"
)

// RepositorySelectorModel represents the repository selection view
type RepositorySelectorModel struct {
	input              textinput.Model
	repos              []models.Repository
	filteredRepos      []models.Repository
	selectedIndex      int
	project            models.Project
	item               models.ProjectItem
	width              int
	height             int
	saveAsDefault      bool // Toggle to save repository as default
}

func NewRepositorySelectorModel(repos []models.Repository, project models.Project, item models.ProjectItem) RepositorySelectorModel {
	ti := textinput.New()
	ti.Placeholder = "Type to filter repositories..."
	ti.Focus()
	ti.Width = 80

	return RepositorySelectorModel{
		input:         ti,
		repos:         repos,
		filteredRepos: repos, // Show all initially
		selectedIndex: 0,
		project:       project,
		item:          item,
	}
}

func (m RepositorySelectorModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m RepositorySelectorModel) Update(msg tea.Msg) (RepositorySelectorModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		inputWidth := msg.Width - 10
		if inputWidth < 40 {
			inputWidth = 40
		}
		m.input.Width = inputWidth
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "down", "ctrl+n":
			if len(m.filteredRepos) > 0 {
				m.selectedIndex++
				if m.selectedIndex >= len(m.filteredRepos) {
					m.selectedIndex = 0
				}
			}
			return m, nil
			
		case "up", "ctrl+p":
			if len(m.filteredRepos) > 0 {
				m.selectedIndex--
				if m.selectedIndex < 0 {
					m.selectedIndex = len(m.filteredRepos) - 1
				}
			}
			return m, nil
			
		case "enter":
			if len(m.filteredRepos) > 0 && m.selectedIndex >= 0 && m.selectedIndex < len(m.filteredRepos) {
				return m, ConvertDraftCmd(m.project, m.item, m.filteredRepos[m.selectedIndex], m.saveAsDefault)
			}
			return m, nil
			
		case "ctrl+d":
			// Toggle save as default
			m.saveAsDefault = !m.saveAsDefault
			return m, nil
		}
	}

	// Update the input and filter repositories
	oldValue := m.input.Value()
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	
	// If the filter text changed, update filtered repos
	if m.input.Value() != oldValue {
		m.filterRepositories()
		m.selectedIndex = 0 // Reset selection when filter changes
	}
	
	return m, cmd
}

func (m *RepositorySelectorModel) filterRepositories() {
	filter := strings.ToLower(m.input.Value())
	
	if filter == "" {
		m.filteredRepos = m.repos
		return
	}
	
	m.filteredRepos = []models.Repository{}
	for _, repo := range m.repos {
		repoName := strings.ToLower(repo.Name)
		repoOwner := strings.ToLower(repo.Owner)
		repoDesc := strings.ToLower(repo.Description)
		fullName := strings.ToLower(fmt.Sprintf("%s/%s", repo.Owner, repo.Name))
		
		if strings.Contains(repoName, filter) ||
			strings.Contains(repoOwner, filter) ||
			strings.Contains(fullName, filter) ||
			strings.Contains(repoDesc, filter) {
			m.filteredRepos = append(m.filteredRepos, repo)
		}
	}
}

func (m RepositorySelectorModel) View() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4")).
		Padding(1, 2)

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		MarginLeft(2)

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#626262")).
		MarginLeft(2).
		MarginTop(1)

	itemStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#5555FF")).
		Padding(0, 1)

	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#7D56F4")).
		Bold(true).
		Padding(0, 1)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7D56F4")).
		MarginLeft(2).
		MarginTop(1)

	var b strings.Builder

	// Header
	b.WriteString(titleStyle.Render(fmt.Sprintf("Convert \"%s\" to issue", m.item.Title)))
	b.WriteString("\n")
	
	// Input
	b.WriteString(labelStyle.Render("Select repository:"))
	b.WriteString("\n")
	b.WriteString("  " + m.input.View())
	b.WriteString("\n")

	// Filtered repositories dropdown
	if len(m.filteredRepos) > 0 {
		var dropdown strings.Builder
		
		// Show up to 10 repositories
		maxShow := 10
		if len(m.filteredRepos) < maxShow {
			maxShow = len(m.filteredRepos)
		}
		
		for i := 0; i < maxShow; i++ {
			repo := m.filteredRepos[i]
			visibility := "🔒"
			if !repo.IsPrivate {
				visibility = "🔓"
			}
			repoText := fmt.Sprintf("%s %s/%s", visibility, repo.Owner, repo.Name)
			
			if i == m.selectedIndex {
				dropdown.WriteString(selectedStyle.Render("▸ " + repoText))
			} else {
				dropdown.WriteString(itemStyle.Render("  " + repoText))
			}
			
			if i < maxShow-1 {
				dropdown.WriteString("\n")
			}
		}
		
		if len(m.filteredRepos) > maxShow {
			dropdown.WriteString("\n")
			dropdown.WriteString(labelStyle.Render(fmt.Sprintf("  ... and %d more", len(m.filteredRepos)-maxShow)))
		}
		
		b.WriteString(boxStyle.Render(dropdown.String()))
		b.WriteString("\n")
	} else {
		noResultsStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			MarginLeft(2).
			MarginTop(1)
		b.WriteString(noResultsStyle.Render("No repositories found"))
		b.WriteString("\n")
	}

	// Save as default toggle
	toggleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00D7FF")).
		MarginLeft(2).
		MarginTop(1)
	
	checkbox := "[ ]"
	if m.saveAsDefault {
		checkbox = "[✓]"
	}
	b.WriteString(toggleStyle.Render(fmt.Sprintf("%s Save as default repository for this project", checkbox)))
	b.WriteString("\n")

	// Help
	help := "↑/↓: navigate • enter: select • ctrl+d: toggle default • esc: cancel"
	b.WriteString(helpStyle.Render(help))

	return b.String()
}

// ConvertDraftCmd signals draft conversion
func ConvertDraftCmd(project models.Project, item models.ProjectItem, repo models.Repository, saveAsDefault bool) tea.Cmd {
	return func() tea.Msg {
		return ConvertDraftMsg{
			Project:       project,
			Item:          item,
			Repository:    repo,
			SaveAsDefault: saveAsDefault,
		}
	}
}

// ConvertDraftMsg is sent when converting a draft to an issue
type ConvertDraftMsg struct {
	Project       models.Project
	Item          models.ProjectItem
	Repository    models.Repository
	SaveAsDefault bool
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
