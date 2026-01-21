package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/thomaskoefod/githubProjectTUI/internal/models"
)

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

// ProjectDetailModel represents the project detail view
type ProjectDetailModel struct {
	project models.Project
	items   []models.ProjectItem
	table   table.Model
	width   int
	height  int
}

func NewProjectDetailModel(project models.Project, items []models.ProjectItem) ProjectDetailModel {
	columns := []table.Column{
		{Title: "Type", Width: 12},
		{Title: "Title", Width: 40},
		{Title: "Assignees", Width: 20},
		{Title: "Status", Width: 12},
		{Title: "Number", Width: 10},
	}

	rows := make([]table.Row, len(items))
	for i, item := range items {
		itemType := item.Type
		if itemType == "" {
			itemType = "Unknown"
		}
		
		status := item.State
		if status == "" {
			status = "-"
		}

		number := "-"
		if item.Number > 0 {
			number = fmt.Sprintf("#%d", item.Number)
		}

		// Format assignees as comma-separated list with @ prefix
		assignees := "-"
		if len(item.Assignees) > 0 {
			assigneeList := make([]string, len(item.Assignees))
			for j, a := range item.Assignees {
				assigneeList[j] = "@" + a
			}
			assignees = strings.Join(assigneeList, ", ")
			assignees = truncate(assignees, 20)
		}

		rows[i] = table.Row{
			itemType,
			truncate(item.Title, 40),
			assignees,
			status,
			number,
		}
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(20),  // Will be updated on WindowSizeMsg
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	return ProjectDetailModel{
		project: project,
		items:   items,
		table:   t,
	}
}

func (m ProjectDetailModel) Init() tea.Cmd {
	return nil
}

func (m ProjectDetailModel) Update(msg tea.Msg) (ProjectDetailModel, tea.Cmd) {
	var cmd tea.Cmd
	
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Adjust table height based on window size
		// Leave room for header (6 lines) and footer (2 lines)
		tableHeight := msg.Height - 8
		if tableHeight < 5 {
			tableHeight = 5
		}
		m.table.SetHeight(tableHeight)
		
		// Adjust column widths based on terminal width
		availableWidth := msg.Width - 10
		if availableWidth < 60 {
			availableWidth = 60
		}
		
		// Calculate column widths proportionally
		typeWidth := 12
		numberWidth := 10
		statusWidth := 12
		assigneesWidth := 20
		titleWidth := availableWidth - typeWidth - numberWidth - statusWidth - assigneesWidth - 10
		if titleWidth < 20 {
			titleWidth = 20
		}
		
		m.table.SetColumns([]table.Column{
			{Title: "Type", Width: typeWidth},
			{Title: "Title", Width: titleWidth},
			{Title: "Assignees", Width: assigneesWidth},
			{Title: "Status", Width: statusWidth},
			{Title: "Number", Width: numberWidth},
		})
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "n":
			// Create new item
			return m, CreateItemCmd(m.project)
		case "e":
			// Edit selected item
			if m.table.Cursor() < len(m.items) {
				return m, EditItemCmd(m.items[m.table.Cursor()])
			}
		case "d":
			// Delete selected item
			if m.table.Cursor() < len(m.items) {
				return m, DeleteItemCmd(m.project, m.items[m.table.Cursor()])
			}
		case "enter":
			// View item details
			if m.table.Cursor() < len(m.items) {
				return m, ViewItemCmd(m.project, m.items[m.table.Cursor()])
			}
		}
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m ProjectDetailModel) View() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4")).
		MarginLeft(2).
		MarginTop(1)

	infoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		MarginLeft(2)

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#626262")).
		MarginLeft(2).
		MarginBottom(1)

	var b strings.Builder

	// Title and project info
	b.WriteString(titleStyle.Render(m.project.Title))
	b.WriteString("\n")

	if m.project.ShortDescription != "" {
		b.WriteString(infoStyle.Render(m.project.ShortDescription))
		b.WriteString("\n")
	}

	status := "Open"
	if m.project.Closed {
		status = "Closed"
	}
	visibility := "Private"
	if m.project.Public {
		visibility = "Public"
	}
	
	b.WriteString(infoStyle.Render(fmt.Sprintf("%s • %s • %d items", 
		status, visibility, len(m.items))))
	b.WriteString("\n\n")

	// Items table
	b.WriteString(m.table.View())
	b.WriteString("\n\n")

	// Help
	b.WriteString(helpStyle.Render("enter: view • n: new item • e: edit • d: delete • esc: back • q: quit"))

	return b.String()
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

// CreateItemCmd signals item creation
func CreateItemCmd(project models.Project) tea.Cmd {
	return func() tea.Msg {
		return CreateItemMsg{Project: project}
	}
}

// ViewItemCmd signals viewing an item
func ViewItemCmd(project models.Project, item models.ProjectItem) tea.Cmd {
	return func() tea.Msg {
		return ViewItemMsg{Project: project, Item: item}
	}
}

// EditItemCmd signals item editing
func EditItemCmd(item models.ProjectItem) tea.Cmd {
	return func() tea.Msg {
		return EditItemMsg{Item: item}
	}
}

// DeleteItemCmd signals item deletion
func DeleteItemCmd(project models.Project, item models.ProjectItem) tea.Cmd {
	return func() tea.Msg {
		return DeleteItemMsg{Project: project, Item: item}
	}
}

// CreateItemMsg is sent to create a new item
type CreateItemMsg struct {
	Project models.Project
}

// ViewItemMsg is sent to view an item's details
type ViewItemMsg struct {
	Project models.Project
	Item    models.ProjectItem
}

// EditItemMsg is sent to edit an item
type EditItemMsg struct {
	Item models.ProjectItem
}

// DeleteItemMsg is sent to delete an item
type DeleteItemMsg struct {
	Project models.Project
	Item    models.ProjectItem
}
