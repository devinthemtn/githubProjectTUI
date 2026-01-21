package ui

import (
	"fmt"
	"os/exec"
	"runtime"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/thomaskoefod/githubProjectTUI/internal/api"
	"github.com/thomaskoefod/githubProjectTUI/internal/auth"
	"github.com/thomaskoefod/githubProjectTUI/internal/models"
)

type view int

const (
	viewLoading view = iota
	viewOwnerSelector
	viewProjectList
	viewProjectDetail
	viewItemDetail
	viewItemEditor
	viewProjectCreator
	viewHelp
)

type Model struct {
	currentView     view
	apiClient       *api.Client
	username        string
	orgs            []string
	currentOwner    string
	currentIsUser   bool
	ownerSelector   OwnerSelectorModel
	projectList     ProjectListModel
	projectDetail   ProjectDetailModel
	itemDetail      ItemDetailModel
	itemEditor      ItemEditorModel
	projectCreator  ProjectCreatorModel
	width           int
	height          int
	err             error
	loading         bool
	message         string
	debugMode       bool
}

func NewModel() Model {
	return Model{
		currentView: viewLoading,
		loading:     true,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		initializeApp,
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Update all sub-models
		switch m.currentView {
		case viewOwnerSelector:
			m.ownerSelector, _ = m.ownerSelector.Update(msg)
		case viewProjectList:
			m.projectList, _ = m.projectList.Update(msg)
		case viewProjectDetail:
			m.projectDetail, _ = m.projectDetail.Update(msg)
		case viewItemDetail:
			m.itemDetail, _ = m.itemDetail.Update(msg)
		case viewItemEditor:
			m.itemEditor, _ = m.itemEditor.Update(msg)
		case viewProjectCreator:
			m.projectCreator, _ = m.projectCreator.Update(msg)
		}

		return m, nil

	case InitializedMsg:
		m.apiClient = msg.Client
		m.username = msg.Username
		m.orgs = msg.Orgs
		m.loading = false
		
		// Show owner selector if there are orgs, otherwise go straight to projects
		if len(msg.Orgs) > 0 {
			m.ownerSelector = NewOwnerSelectorModel(msg.Username, msg.Orgs)
			m.ownerSelector.width = m.width
			m.ownerSelector.height = m.height
			m.currentView = viewOwnerSelector
			// Force window size update to selector
			m.ownerSelector, _ = m.ownerSelector.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
			return m, nil
		} else {
			m.currentOwner = msg.Username
			m.currentIsUser = true
			return m, loadProjects(m.apiClient, msg.Username, true)
		}

	case OwnerSelectedMsg:
		m.currentOwner = msg.Owner
		m.currentIsUser = msg.IsUser
		m.loading = true
		m.message = fmt.Sprintf("Loading projects for %s...", msg.Owner)
		return m, loadProjects(m.apiClient, msg.Owner, msg.IsUser)

	case ProjectsLoadedMsg:
		m.projectList = NewProjectListModel(msg.Projects)
		m.projectList.width = m.width
		m.projectList.height = m.height
		m.currentView = viewProjectList
		m.loading = false
		// Force window size update to list
		m.projectList, _ = m.projectList.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		return m, nil

	case ProjectSelectedMsg:
		m.loading = true
		m.message = "Loading project items..."
		return m, loadProjectItems(m.apiClient, msg.Project)

	case ProjectItemsLoadedMsg:
		m.projectDetail = NewProjectDetailModel(msg.Project, msg.Items)
		m.projectDetail.width = m.width
		m.projectDetail.height = m.height
		m.currentView = viewProjectDetail
		m.loading = false
		return m, nil

	case CreateItemMsg:
		m.itemEditor = NewItemEditorModel(msg.Project, nil)
		m.itemEditor.width = m.width
		m.itemEditor.height = m.height
		m.currentView = viewItemEditor
		return m, m.itemEditor.Init()

	case ViewItemMsg:
		m.itemDetail = NewItemDetailModel(msg.Project, msg.Item)
		m.itemDetail.width = m.width
		m.itemDetail.height = m.height
		m.currentView = viewItemDetail
		return m, nil

	case NewProjectMsg:
		m.projectCreator = NewProjectCreatorModel(m.currentOwner, m.currentIsUser)
		m.projectCreator.width = m.width
		m.projectCreator.height = m.height
		m.currentView = viewProjectCreator
		return m, m.projectCreator.Init()

	case EditItemMsg:
		m.itemEditor = NewItemEditorModel(models.Project{}, &msg.Item)
		m.itemEditor.width = m.width
		m.itemEditor.height = m.height
		m.currentView = viewItemEditor
		return m, m.itemEditor.Init()

	case SaveItemMsg:
		m.loading = true
		m.message = "Saving item..."
		return m, saveItem(m.apiClient, msg)

	case OpenURLMsg:
		// Open URL in default browser
		return m, openURL(msg.URL)

	case CreateProjectMsg:
		m.loading = true
		m.message = "Creating project..."
		return m, createProject(m.apiClient, msg)

	case ProjectCreatedMsg:
		m.loading = false
		// Reload projects for current owner
		return m, loadProjects(m.apiClient, m.currentOwner, m.currentIsUser)

	case ItemSavedMsg:
		m.loading = false
		// Reload project items
		return m, loadProjectItems(m.apiClient, m.itemEditor.project)

	case ErrorMsg:
		m.err = msg.Err
		m.loading = false
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.currentView == viewOwnerSelector || m.currentView == viewProjectList {
				return m, tea.Quit
			}
		case "esc":
			// Clear error first
			if m.err != nil {
				m.err = nil
				return m, nil
			}
			
			// Navigate back
			switch m.currentView {
			case viewProjectList:
				if len(m.orgs) > 0 {
					m.currentView = viewOwnerSelector
					return m, nil
				}
			case viewProjectDetail:
				m.currentView = viewProjectList
				return m, nil
			case viewItemDetail:
				m.currentView = viewProjectDetail
				return m, nil
			case viewItemEditor:
				// Go back to item detail if we came from there, otherwise project detail
				if m.itemEditor.item != nil {
					m.currentView = viewItemDetail
				} else {
					m.currentView = viewProjectDetail
				}
				return m, nil
			case viewProjectCreator:
				m.currentView = viewProjectList
				return m, nil
			case viewHelp:
				m.currentView = viewProjectList
				return m, nil
			}
		case "?":
			if m.currentView == viewHelp {
				m.currentView = viewProjectList
			} else {
				m.currentView = viewHelp
			}
			return m, nil
		case "ctrl+d":
			m.debugMode = !m.debugMode
			return m, nil
		}
	}

	// Delegate to sub-models
	var cmd tea.Cmd
	switch m.currentView {
	case viewOwnerSelector:
		m.ownerSelector, cmd = m.ownerSelector.Update(msg)
	case viewProjectList:
		m.projectList, cmd = m.projectList.Update(msg)
	case viewProjectDetail:
		m.projectDetail, cmd = m.projectDetail.Update(msg)
	case viewItemDetail:
		m.itemDetail, cmd = m.itemDetail.Update(msg)
	case viewItemEditor:
		m.itemEditor, cmd = m.itemEditor.Update(msg)
	case viewProjectCreator:
		m.projectCreator, cmd = m.projectCreator.Update(msg)
	}

	return m, cmd
}

func (m Model) View() string {
	if m.loading {
		return m.renderLoading()
	}

	if m.err != nil {
		return m.renderError()
	}

	switch m.currentView {
	case viewOwnerSelector:
		return m.ownerSelector.View()
	case viewProjectList:
		return m.projectList.View()
	case viewProjectDetail:
		return m.projectDetail.View()
	case viewItemDetail:
		return m.itemDetail.View()
	case viewItemEditor:
		return m.itemEditor.View()
	case viewProjectCreator:
		return m.projectCreator.View()
	case viewHelp:
		return m.renderHelp()
	default:
		return "Loading..."
	}
}

func (m Model) renderLoading() string {
	loadingStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7D56F4")).
		Padding(2, 4)

	msg := m.message
	if msg == "" {
		msg = "Initializing..."
	}

	return loadingStyle.Render(msg)
}

func (m Model) renderHelp() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4")).
		Padding(1, 2)

	helpStyle := lipgloss.NewStyle().
		Padding(1, 2)

	title := titleStyle.Render("Keyboard Shortcuts")

	helpText := `
Navigation:
  j/k or ↓/↑     Navigate up/down
  enter          Select item
  esc            Go back

Actions:
  n              Create new item/project
  e              Edit selected item
  d              Delete selected item

General:
  ?              Toggle help
  q or ctrl+c    Quit (from main views)
  ctrl+d         Toggle debug mode
`

	help := helpStyle.Render(helpText)

	return lipgloss.JoinVertical(lipgloss.Left, title, help)
}

func (m Model) renderError() string {
	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF0000")).
		Padding(1, 2)

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#626262")).
		Padding(1, 2)

	debugStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Padding(1, 2)

	var content string
	if m.debugMode {
		content = lipgloss.JoinVertical(lipgloss.Left,
			errorStyle.Render(fmt.Sprintf("Error: %v", m.err)),
			debugStyle.Render(fmt.Sprintf("Current view: %v\nUsername: %s\nOrgs: %v\nOwner: %s", 
				m.currentView, m.username, m.orgs, m.currentOwner)),
			helpStyle.Render("Press esc to continue, q to quit, ctrl+d to hide debug"),
		)
	} else {
		content = lipgloss.JoinVertical(lipgloss.Left,
			errorStyle.Render(fmt.Sprintf("Error: %v", m.err)),
			helpStyle.Render("Press esc to continue, q to quit, ctrl+d for debug"),
		)
	}

	return content
}

// Commands and messages

func initializeApp() tea.Msg {
	// Check authentication
	if err := auth.CheckAuthentication(); err != nil {
		return ErrorMsg{Err: err}
	}

	// Get username
	username, err := auth.GetAuthenticatedUser()
	if err != nil {
		return ErrorMsg{Err: fmt.Errorf("failed to get user: %w", err)}
	}

	// Create API client
	client, err := api.NewClient()
	if err != nil {
		return ErrorMsg{Err: fmt.Errorf("failed to create API client: %w", err)}
	}

	// Get organizations
	orgs, err := client.GetUserOrganizations(username)
	if err != nil {
		// Non-fatal, just log and continue
		orgs = []string{}
	}

	return InitializedMsg{
		Client:   client,
		Username: username,
		Orgs:     orgs,
	}
}

func loadProjects(client *api.Client, owner string, isUser bool) tea.Cmd {
	return func() tea.Msg {
		var projects []models.Project
		var err error
		
		if isUser {
			projects, err = client.ListUserProjects(owner, 100)
		} else {
			projects, err = client.ListOrgProjects(owner, 100)
		}
		
		if err != nil {
			return ErrorMsg{Err: fmt.Errorf("failed to load projects for %s: %w", owner, err)}
		}
		return ProjectsLoadedMsg{Projects: projects}
	}
}

func loadProjectItems(client *api.Client, project models.Project) tea.Cmd {
	return func() tea.Msg {
		items, err := client.ListProjectItems(project.ID, 100)
		if err != nil {
			return ErrorMsg{Err: fmt.Errorf("failed to load items: %w", err)}
		}
		return ProjectItemsLoadedMsg{
			Project: project,
			Items:   items,
		}
	}
}

func saveItem(client *api.Client, msg SaveItemMsg) tea.Cmd {
	return func() tea.Msg {
		if msg.IsNewItem {
			_, err := client.CreateDraftIssue(models.CreateItemInput{
				ProjectID: msg.Project.ID,
				Title:     msg.Title,
				Body:      msg.Body,
			})
			if err != nil {
				return ErrorMsg{Err: fmt.Errorf("failed to create item: %w", err)}
			}
		} else {
			_, err := client.UpdateDraftIssue(msg.Item.ID, msg.Title, msg.Body)
			if err != nil {
				return ErrorMsg{Err: fmt.Errorf("failed to update item: %w", err)}
			}
		}
		return ItemSavedMsg{}
	}
}

func openURL(url string) tea.Cmd {
	return func() tea.Msg {
		// Try different commands based on OS
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("open", url)
		case "linux":
			cmd = exec.Command("xdg-open", url)
		case "windows":
			cmd = exec.Command("cmd", "/c", "start", url)
		default:
			return ErrorMsg{Err: fmt.Errorf("unsupported platform for opening URLs")}
		}
		
		err := cmd.Start()
		if err != nil {
			return ErrorMsg{Err: fmt.Errorf("failed to open URL: %w", err)}
		}
		return nil
	}
}

func createProject(client *api.Client, msg CreateProjectMsg) tea.Cmd {
	return func() tea.Msg {
		// First, get the owner ID
		var ownerID string
		var err error
		
		if msg.IsUserOwner {
			// For user, we need to query the user's node ID
			ownerID, err = client.GetUserNodeID(msg.OwnerLogin)
		} else {
			// For org, we need to query the org's node ID
			ownerID, err = client.GetOrgNodeID(msg.OwnerLogin)
		}
		
		if err != nil {
			return ErrorMsg{Err: fmt.Errorf("failed to get owner ID: %w", err)}
		}
		
		_, err = client.CreateProject(models.CreateProjectInput{
			OwnerID:          ownerID,
			Title:            msg.Title,
			ShortDescription: msg.Description,
			Public:           msg.Public,
		})
		
		if err != nil {
			return ErrorMsg{Err: fmt.Errorf("failed to create project: %w", err)}
		}
		
		return ProjectCreatedMsg{}
	}
}

// Messages

type InitializedMsg struct {
	Client   *api.Client
	Username string
	Orgs     []string
}

type ProjectsLoadedMsg struct {
	Projects []models.Project
}

type ProjectItemsLoadedMsg struct {
	Project models.Project
	Items   []models.ProjectItem
}

type ItemSavedMsg struct{}

type ProjectCreatedMsg struct{}

type ErrorMsg struct {
	Err error
}
