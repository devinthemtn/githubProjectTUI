package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/thomaskoefod/githubProjectTUI/internal/models"
)

// ItemEditorModel represents the item editor view
type ItemEditorModel struct {
	project           models.Project
	owner             string // Owner login (user or org)
	isOrgProject      bool   // True if project belongs to org
	item              *models.ProjectItem
	titleInput        textinput.Model
	bodyInput         textarea.Model
	assigneeInput     textinput.Model
	focusIndex        int
	isNewItem         bool
	width             int
	height            int
	suggestions       []string
	selectedSuggestion int
	showSuggestions   bool
}

func NewItemEditorModel(project models.Project, owner string, isOrgProject bool, item *models.ProjectItem) ItemEditorModel {
	ti := textinput.New()
	ti.Placeholder = "Item title"
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 80  // Will be adjusted on WindowSizeMsg

	ta := textarea.New()
	ta.Placeholder = "Item description (optional)"
	ta.CharLimit = 2000
	ta.SetWidth(80)  // Will be adjusted on WindowSizeMsg
	ta.SetHeight(10)

	ai := textinput.New()
	ai.Placeholder = "Assignee username (optional)"
	ai.CharLimit = 100
	ai.Width = 80  // Will be adjusted on WindowSizeMsg

	isNew := item == nil
	if item != nil {
		ti.SetValue(item.Title)
		ta.SetValue(item.Body)
		if len(item.Assignees) > 0 {
			ai.SetValue(item.Assignees[0])
		}
	}

	return ItemEditorModel{
		project:       project,
		owner:         owner,
		isOrgProject:  isOrgProject,
		item:          item,
		titleInput:    ti,
		bodyInput:     ta,
		assigneeInput: ai,
		focusIndex:    0,
		isNewItem:     isNew,
	}
}

func (m ItemEditorModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m ItemEditorModel) Update(msg tea.Msg) (ItemEditorModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Adjust input widths based on terminal
		inputWidth := msg.Width - 10
		if inputWidth < 40 {
			inputWidth = 40
		}
		m.titleInput.Width = inputWidth
		m.bodyInput.SetWidth(inputWidth)
		m.assigneeInput.Width = inputWidth
		
		// Adjust textarea height
		textareaHeight := msg.Height - 18
		if textareaHeight < 5 {
			textareaHeight = 5
		}
		m.bodyInput.SetHeight(textareaHeight)
		return m, nil

	case tea.KeyMsg:
		// Handle suggestion navigation when assignee field is focused and suggestions are shown
		if m.focusIndex == 2 && m.showSuggestions && len(m.suggestions) > 0 {
			switch msg.String() {
			case "down", "ctrl+n":
				m.selectedSuggestion++
				if m.selectedSuggestion >= len(m.suggestions) {
					m.selectedSuggestion = 0
				}
				return m, nil
			case "up", "ctrl+p":
				m.selectedSuggestion--
				if m.selectedSuggestion < 0 {
					m.selectedSuggestion = len(m.suggestions) - 1
				}
				return m, nil
			case "enter":
				// Select the suggestion
				if m.selectedSuggestion >= 0 && m.selectedSuggestion < len(m.suggestions) {
					m.assigneeInput.SetValue(m.suggestions[m.selectedSuggestion])
					m.showSuggestions = false
					m.suggestions = nil
				}
				return m, nil
			case "esc":
				// Close suggestions without selecting
				m.showSuggestions = false
				m.suggestions = nil
				return m, nil
			}
		}

		switch msg.String() {
		case "ctrl+s":
			// Save
			return m, m.saveCmd()
		case "tab", "shift+tab":
			// Switch focus
			if msg.String() == "tab" {
				m.focusIndex++
			} else {
				m.focusIndex--
			}

			if m.focusIndex > 2 {
				m.focusIndex = 0
			} else if m.focusIndex < 0 {
				m.focusIndex = 2
			}

			m.titleInput.Blur()
			m.bodyInput.Blur()
			m.assigneeInput.Blur()
			m.showSuggestions = false

			switch m.focusIndex {
			case 0:
				m.titleInput.Focus()
			case 1:
				m.bodyInput.Focus()
			case 2:
				m.assigneeInput.Focus()
			}

			return m, nil
		}

	case UserSuggestionsMsg:
		m.suggestions = msg.Users
		m.selectedSuggestion = 0
		m.showSuggestions = len(msg.Users) > 0
		return m, nil
	}

	var cmd tea.Cmd
	switch m.focusIndex {
	case 0:
		m.titleInput, cmd = m.titleInput.Update(msg)
		cmds = append(cmds, cmd)
	case 1:
		m.bodyInput, cmd = m.bodyInput.Update(msg)
		cmds = append(cmds, cmd)
	case 2:
		oldValue := m.assigneeInput.Value()
		m.assigneeInput, cmd = m.assigneeInput.Update(msg)
		cmds = append(cmds, cmd)
		
		// If value changed and not empty, search for users
		newValue := m.assigneeInput.Value()
		if newValue != oldValue && len(newValue) >= 2 {
			cmds = append(cmds, searchUsersCmd(newValue, m.owner, m.isOrgProject))
		} else if newValue == "" {
			m.showSuggestions = false
			m.suggestions = nil
		}
	}

	return m, tea.Batch(cmds...)
}

func (m ItemEditorModel) View() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4")).
		MarginLeft(2).
		MarginTop(1)

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		MarginLeft(2)

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#626262")).
		MarginLeft(2).
		MarginTop(1)

	var b strings.Builder

	title := "New Item"
	if !m.isNewItem {
		title = "Edit Item"
	}
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n")
	b.WriteString(labelStyle.Render("Project: " + m.project.Title))
	b.WriteString("\n\n")

	b.WriteString(labelStyle.Render("Title:"))
	b.WriteString("\n")
	b.WriteString("  " + m.titleInput.View())
	b.WriteString("\n\n")

	b.WriteString(labelStyle.Render("Description:"))
	b.WriteString("\n")
	b.WriteString("  " + m.bodyInput.View())
	b.WriteString("\n\n")

	b.WriteString(labelStyle.Render("Assignee:"))
	b.WriteString("\n")
	b.WriteString("  " + m.assigneeInput.View())
	b.WriteString("\n")

	// Show suggestions dropdown if available
	if m.showSuggestions && len(m.suggestions) > 0 {
		suggestionStyle := lipgloss.NewStyle().
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
			MarginLeft(2)

		var suggestions strings.Builder
		for i, user := range m.suggestions {
			if i == m.selectedSuggestion {
				suggestions.WriteString(selectedStyle.Render("▸ @" + user))
			} else {
				suggestions.WriteString(suggestionStyle.Render("  @" + user))
			}
			if i < len(m.suggestions)-1 {
				suggestions.WriteString("\n")
			}
		}

		b.WriteString(boxStyle.Render(suggestions.String()))
		b.WriteString("\n")
	}

	helpText := "tab: switch fields • ctrl+s: save • esc: cancel"
	if m.showSuggestions && len(m.suggestions) > 0 {
		helpText = "↑/↓: navigate suggestions • enter: select • esc: close • ctrl+s: save"
	}
	b.WriteString(helpStyle.Render(helpText))

	return b.String()
}

func (m ItemEditorModel) saveCmd() tea.Cmd {
	return func() tea.Msg {
		return SaveItemMsg{
			Project:   m.project,
			Item:      m.item,
			Title:     m.titleInput.Value(),
			Body:      m.bodyInput.Value(),
			Assignee:  m.assigneeInput.Value(),
			IsNewItem: m.isNewItem,
		}
	}
}

// SaveItemMsg is sent when saving an item
type SaveItemMsg struct {
	Project   models.Project
	Item      *models.ProjectItem
	Title     string
	Body      string
	Assignee  string
	IsNewItem bool
}

// UserSuggestionsMsg contains user search results
type UserSuggestionsMsg struct {
Users []string
}
