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
	project     models.Project
	item        *models.ProjectItem
	titleInput  textinput.Model
	bodyInput   textarea.Model
	focusIndex  int
	isNewItem   bool
	width       int
	height      int
}

func NewItemEditorModel(project models.Project, item *models.ProjectItem) ItemEditorModel {
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

	isNew := item == nil
	if item != nil {
		ti.SetValue(item.Title)
		ta.SetValue(item.Body)
	}

	return ItemEditorModel{
		project:     project,
		item:        item,
		titleInput:  ti,
		bodyInput:   ta,
		focusIndex:  0,
		isNewItem:   isNew,
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
		
		// Adjust textarea height
		textareaHeight := msg.Height - 15
		if textareaHeight < 5 {
			textareaHeight = 5
		}
		m.bodyInput.SetHeight(textareaHeight)
		return m, nil

	case tea.KeyMsg:
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

			if m.focusIndex > 1 {
				m.focusIndex = 0
			} else if m.focusIndex < 0 {
				m.focusIndex = 1
			}

			if m.focusIndex == 0 {
				m.titleInput.Focus()
				m.bodyInput.Blur()
			} else {
				m.titleInput.Blur()
				m.bodyInput.Focus()
			}

			return m, nil
		}
	}

	var cmd tea.Cmd
	if m.focusIndex == 0 {
		m.titleInput, cmd = m.titleInput.Update(msg)
		cmds = append(cmds, cmd)
	} else {
		m.bodyInput, cmd = m.bodyInput.Update(msg)
		cmds = append(cmds, cmd)
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
	b.WriteString("\n")

	b.WriteString(helpStyle.Render("tab: switch fields • ctrl+s: save • esc: cancel"))

	return b.String()
}

func (m ItemEditorModel) saveCmd() tea.Cmd {
	return func() tea.Msg {
		return SaveItemMsg{
			Project:   m.project,
			Item:      m.item,
			Title:     m.titleInput.Value(),
			Body:      m.bodyInput.Value(),
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
	IsNewItem bool
}
