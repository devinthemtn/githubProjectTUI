package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/thomaskoefod/githubProjectTUI/internal/models"
)

var (
	itemDetailTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#7D56F4")).
				MarginLeft(2).
				MarginTop(1)

	itemDetailLabelStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#00D7FF")).
				MarginLeft(2).
				MarginTop(1)

	itemDetailValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				MarginLeft(4)

	itemDetailMetaStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#888888")).
				MarginLeft(2)

	itemDetailHelpStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#626262")).
				MarginLeft(2).
				MarginTop(1)

	itemDetailBoxStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#7D56F4")).
				Padding(1, 2).
				MarginLeft(2).
				MarginRight(2).
				MarginTop(1)
)

// ItemDetailModel represents the item detail view
type ItemDetailModel struct {
	project models.Project
	item    models.ProjectItem
	width   int
	height  int
}

func NewItemDetailModel(project models.Project, item models.ProjectItem) ItemDetailModel {
	return ItemDetailModel{
		project: project,
		item:    item,
	}
}

func (m ItemDetailModel) Init() tea.Cmd {
	return nil
}

func (m ItemDetailModel) Update(msg tea.Msg) (ItemDetailModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "e":
			// Edit item
			return m, EditItemCmd(m.item)
		case "d":
			// Delete item
			return m, DeleteItemCmd(m.project, m.item)
		case "o":
			// Open in browser (if URL exists)
			if m.item.URL != "" {
				return m, OpenURLCmd(m.item.URL)
			}
		}
	}

	return m, nil
}

func (m ItemDetailModel) View() string {
	var b strings.Builder

	// Title
	b.WriteString(itemDetailTitleStyle.Render(m.item.Title))
	b.WriteString("\n")

	// Metadata row
	var metaParts []string
	
	// Type
	itemType := m.item.Type
	if itemType == "" {
		itemType = "Unknown"
	}
	metaParts = append(metaParts, fmt.Sprintf("Type: %s", itemType))

	// State
	if m.item.State != "" {
		metaParts = append(metaParts, fmt.Sprintf("State: %s", m.item.State))
	}

	// Number
	if m.item.Number > 0 {
		metaParts = append(metaParts, fmt.Sprintf("#%d", m.item.Number))
	}

	b.WriteString(itemDetailMetaStyle.Render(strings.Join(metaParts, " • ")))
	b.WriteString("\n")

	// Description
	if m.item.Body != "" {
		b.WriteString(itemDetailLabelStyle.Render("Description:"))
		b.WriteString("\n")
		
		// Use full width for description
		boxWidth := m.width - 10
		if boxWidth < 40 {
			boxWidth = 40
		}
		
		// Word wrap the description to fit terminal
		wrapped := wordWrap(m.item.Body, boxWidth)
		b.WriteString(itemDetailBoxStyle.Width(boxWidth).Render(wrapped))
		b.WriteString("\n")
	} else {
		b.WriteString(itemDetailMetaStyle.Render("(No description)"))
		b.WriteString("\n\n")
	}

	// Timestamps
	b.WriteString(itemDetailLabelStyle.Render("Details:"))
	b.WriteString("\n")
	
	if !m.item.CreatedAt.IsZero() {
		b.WriteString(itemDetailValueStyle.Render(fmt.Sprintf("Created: %s", formatTime(m.item.CreatedAt))))
		b.WriteString("\n")
	}
	
	if !m.item.UpdatedAt.IsZero() {
		b.WriteString(itemDetailValueStyle.Render(fmt.Sprintf("Updated: %s", formatTime(m.item.UpdatedAt))))
		b.WriteString("\n")
	}

	// URL
	if m.item.URL != "" {
		b.WriteString(itemDetailValueStyle.Render(fmt.Sprintf("URL: %s", m.item.URL)))
		b.WriteString("\n")
	}

	// Project context
	b.WriteString("\n")
	b.WriteString(itemDetailMetaStyle.Render(fmt.Sprintf("Project: %s", m.project.Title)))
	b.WriteString("\n")

	// Help
	b.WriteString("\n")
	helpText := "e: edit • d: delete"
	if m.item.URL != "" {
		helpText += " • o: open in browser"
	}
	helpText += " • esc: back • q: quit"
	b.WriteString(itemDetailHelpStyle.Render(helpText))

	return b.String()
}

func formatTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "yesterday"
		}
		return fmt.Sprintf("%d days ago", days)
	default:
		return t.Format("Jan 2, 2006")
	}
}

func wordWrap(text string, width int) string {
	if width <= 0 {
		width = 80
	}

	var result strings.Builder
	var currentLine strings.Builder
	currentLen := 0

	words := strings.Fields(text)
	for i, word := range words {
		wordLen := len(word)
		
		if currentLen+wordLen+1 > width {
			result.WriteString(currentLine.String())
			result.WriteString("\n")
			currentLine.Reset()
			currentLen = 0
		}

		if currentLen > 0 {
			currentLine.WriteString(" ")
			currentLen++
		}

		currentLine.WriteString(word)
		currentLen += wordLen

		if i == len(words)-1 {
			result.WriteString(currentLine.String())
		}
	}

	return result.String()
}

// OpenURLCmd signals opening a URL in browser
func OpenURLCmd(url string) tea.Cmd {
	return func() tea.Msg {
		return OpenURLMsg{URL: url}
	}
}

// OpenURLMsg is sent to open a URL
type OpenURLMsg struct {
	URL string
}
