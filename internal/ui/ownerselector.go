package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ownerItem represents a user or organization in the list
type ownerItem struct {
	login    string
	ownerType string
	isUser   bool
}

func (i ownerItem) FilterValue() string { return i.login }
func (i ownerItem) Title() string {
	if i.isUser {
		return fmt.Sprintf("👤 %s (Personal)", i.login)
	}
	return fmt.Sprintf("🏢 %s (Organization)", i.login)
}
func (i ownerItem) Description() string {
	if i.isUser {
		return "Your personal projects"
	}
	return "Organization projects"
}

// OwnerSelectorModel represents the owner selection view
type OwnerSelectorModel struct {
	list     list.Model
	username string
	orgs     []string
	width    int
	height   int
}

func NewOwnerSelectorModel(username string, orgs []string) OwnerSelectorModel {
	items := []list.Item{
		ownerItem{login: username, ownerType: "User", isUser: true},
	}
	
	for _, org := range orgs {
		items = append(items, ownerItem{login: org, ownerType: "Organization", isUser: false})
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = ""  // Remove title to save space
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)  // We'll show custom help
	l.Styles.Title = lipgloss.NewStyle()

	return OwnerSelectorModel{
		list:     l,
		username: username,
		orgs:     orgs,
	}
}

func (m OwnerSelectorModel) Init() tea.Cmd {
	return nil
}

func (m OwnerSelectorModel) Update(msg tea.Msg) (OwnerSelectorModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Use almost full height, leaving room for header and footer
		m.list.SetWidth(msg.Width)
		m.list.SetHeight(msg.Height - 6)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if i, ok := m.list.SelectedItem().(ownerItem); ok {
				return m, SelectOwnerCmd(i.login, i.isUser)
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m OwnerSelectorModel) View() string {
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#626262")).
		Padding(1, 2)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4")).
		Padding(1, 2)

	help := helpStyle.Render("↑/↓: navigate • enter: select • q: quit")
	
	// Show a clear header
	header := titleStyle.Render("Select which projects to view:")
	
	return lipgloss.JoinVertical(lipgloss.Left, header, m.list.View(), help)
}

// SelectOwnerCmd signals owner selection
func SelectOwnerCmd(owner string, isUser bool) tea.Cmd {
	return func() tea.Msg {
		return OwnerSelectedMsg{Owner: owner, IsUser: isUser}
	}
}

// OwnerSelectedMsg is sent when an owner is selected
type OwnerSelectedMsg struct {
	Owner  string
	IsUser bool
}
