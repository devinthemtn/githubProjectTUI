package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	projectCreatorTitleStyle = lipgloss.NewStyle().
					Bold(true).
					Foreground(lipgloss.Color("#7D56F4")).
					MarginLeft(2).
					MarginTop(1)

	projectCreatorLabelStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("#888888")).
					MarginLeft(2)

	projectCreatorHelpStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#626262")).
				MarginLeft(2).
				MarginTop(1)

	projectCreatorErrorStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("#FF0000")).
					MarginLeft(2)
)

// ProjectCreatorModel represents the project creation form
type ProjectCreatorModel struct {
	ownerLogin    string
	isUserOwner   bool
	titleInput    textinput.Model
	descInput     textarea.Model
	publicToggle  bool
	focusIndex    int
	width         int
	height        int
	validationErr string
}

func NewProjectCreatorModel(ownerLogin string, isUserOwner bool) ProjectCreatorModel {
	ti := textinput.New()
	ti.Placeholder = "Project title"
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 80

	ta := textarea.New()
	ta.Placeholder = "Project description (optional)"
	ta.CharLimit = 500
	ta.SetWidth(80)
	ta.SetHeight(5)

	return ProjectCreatorModel{
		ownerLogin:   ownerLogin,
		isUserOwner:  isUserOwner,
		titleInput:   ti,
		descInput:    ta,
		publicToggle: false,
		focusIndex:   0,
	}
}

func (m ProjectCreatorModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m ProjectCreatorModel) Update(msg tea.Msg) (ProjectCreatorModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		
		inputWidth := msg.Width - 10
		if inputWidth < 40 {
			inputWidth = 40
		}
		m.titleInput.Width = inputWidth
		m.descInput.SetWidth(inputWidth)
		
		textareaHeight := msg.Height - 20
		if textareaHeight < 3 {
			textareaHeight = 3
		}
		m.descInput.SetHeight(textareaHeight)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+s":
			// Validate and save
			if m.titleInput.Value() == "" {
				m.validationErr = "Title is required"
				return m, nil
			}
			m.validationErr = ""
			return m, m.createProjectCmd()
			
		case "tab", "shift+tab":
			// Switch focus between fields
			m.validationErr = ""
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

			m.updateFocus()
			return m, nil
			
		case " ":
			// Toggle public/private if focused on toggle
			if m.focusIndex == 2 {
				m.publicToggle = !m.publicToggle
				return m, nil
			}
		}
	}

	var cmd tea.Cmd
	switch m.focusIndex {
	case 0:
		m.titleInput, cmd = m.titleInput.Update(msg)
		cmds = append(cmds, cmd)
	case 1:
		m.descInput, cmd = m.descInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *ProjectCreatorModel) updateFocus() {
	switch m.focusIndex {
	case 0:
		m.titleInput.Focus()
		m.descInput.Blur()
	case 1:
		m.titleInput.Blur()
		m.descInput.Focus()
	case 2:
		m.titleInput.Blur()
		m.descInput.Blur()
	}
}

func (m ProjectCreatorModel) View() string {
	var b strings.Builder

	b.WriteString(projectCreatorTitleStyle.Render("Create New Project"))
	b.WriteString("\n")
	
	ownerType := "Organization"
	if m.isUserOwner {
		ownerType = "User"
	}
	b.WriteString(projectCreatorLabelStyle.Render("Owner: " + m.ownerLogin + " (" + ownerType + ")"))
	b.WriteString("\n\n")

	// Title input
	focusIndicator := " "
	if m.focusIndex == 0 {
		focusIndicator = "▶"
	}
	b.WriteString(projectCreatorLabelStyle.Render(focusIndicator + " Title:"))
	b.WriteString("\n")
	b.WriteString("  " + m.titleInput.View())
	b.WriteString("\n\n")

	// Description input
	focusIndicator = " "
	if m.focusIndex == 1 {
		focusIndicator = "▶"
	}
	b.WriteString(projectCreatorLabelStyle.Render(focusIndicator + " Description:"))
	b.WriteString("\n")
	b.WriteString("  " + m.descInput.View())
	b.WriteString("\n\n")

	// Public/Private toggle
	focusIndicator = " "
	if m.focusIndex == 2 {
		focusIndicator = "▶"
	}
	visibility := "Private"
	checkbox := "[ ]"
	if m.publicToggle {
		visibility = "Public"
		checkbox = "[x]"
	}
	b.WriteString(projectCreatorLabelStyle.Render(focusIndicator + " Visibility: " + checkbox + " " + visibility))
	b.WriteString("\n")

	// Validation error
	if m.validationErr != "" {
		b.WriteString("\n")
		b.WriteString(projectCreatorErrorStyle.Render("⚠ " + m.validationErr))
		b.WriteString("\n")
	}

	// Help
	b.WriteString("\n")
	b.WriteString(projectCreatorHelpStyle.Render("tab: next field • space: toggle visibility • ctrl+s: create • esc: cancel"))

	return b.String()
}

func (m ProjectCreatorModel) createProjectCmd() tea.Cmd {
	return func() tea.Msg {
		return CreateProjectMsg{
			OwnerLogin:  m.ownerLogin,
			IsUserOwner: m.isUserOwner,
			Title:       m.titleInput.Value(),
			Description: m.descInput.Value(),
			Public:      m.publicToggle,
		}
	}
}

// CreateProjectMsg is sent when creating a project
type CreateProjectMsg struct {
	OwnerLogin  string
	IsUserOwner bool
	Title       string
	Description string
	Public      bool
}
