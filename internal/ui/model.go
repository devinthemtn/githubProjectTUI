package ui

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/thomaskoefod/githubProjectTUI/internal/api"
	"github.com/thomaskoefod/githubProjectTUI/internal/auth"
	"github.com/thomaskoefod/githubProjectTUI/internal/config"
	apierrors "github.com/thomaskoefod/githubProjectTUI/internal/errors"
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
	viewRepositorySelector
	viewHelp
)

type Model struct {
	currentView        view
	apiClient          *api.Client
	config             *config.Config
	username           string
	orgs               []string
	currentOwner       string
	currentIsUser      bool
	ownerSelector      OwnerSelectorModel
	projectList        ProjectListModel
	projectDetail      ProjectDetailModel
	itemDetail         ItemDetailModel
	itemEditor         ItemEditorModel
	projectCreator     ProjectCreatorModel
	repositorySelector RepositorySelectorModel
	width              int
	height             int
	err                error
	loading            bool
	message            string
	debugMode          bool
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
		case viewRepositorySelector:
			m.repositorySelector, _ = m.repositorySelector.Update(msg)
		}

		return m, nil

	case InitializedMsg:
		m.apiClient = msg.Client
		m.config = msg.Config
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
		m.itemEditor = NewItemEditorModel(msg.Project, m.currentOwner, !m.currentIsUser, nil)
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
		m.itemEditor = NewItemEditorModel(models.Project{}, m.currentOwner, !m.currentIsUser, &msg.Item)
		m.itemEditor.width = m.width
		m.itemEditor.height = m.height
		m.currentView = viewItemEditor
		return m, m.itemEditor.Init()

	case SaveItemMsg:
		m.loading = true
		m.message = "Saving item..."
		return m, saveItem(m.apiClient, msg)

	case SaveAndConvertMsg:
		m.loading = true
		m.message = "Saving and preparing conversion..."
		return m, saveAndConvert(m.apiClient, m.currentOwner, m.currentIsUser, msg)

	case ItemSavedAndReadyToConvertMsg:
		// Item saved, now show repository selector
		m.loading = false
		return m, LoadRepositoriesCmd(msg.Project, msg.Item)

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

	case PartialSuccessMsg:
		m.loading = false
		m.err = nil
		// Show warning message with success indicator
		m.message = "⚠️ " + msg.Message
		// Reload project items but keep warning visible
		return m, loadProjectItems(m.apiClient, m.itemEditor.project)

	case DeleteItemMsg:
		m.loading = true
		m.message = "Deleting item..."
		return m, deleteItem(m.apiClient, msg)

	case ItemDeletedMsg:
		m.loading = false
		m.message = ""
		// Reload project items to reflect deletion
		return m, loadProjectItems(m.apiClient, msg.Project)

	case LoadRepositoriesMsg:
		m.loading = true
		m.message = "Loading repositories..."
		return m, loadRepositories(m.apiClient, m.currentOwner, m.currentIsUser, msg.Project, msg.Item)

	case RepositoriesLoadedMsg:
		// Check if there's a saved default repository for this project (if config is available)
		if m.config != nil {
			if defaultRepoID, ok := m.config.GetDefaultRepository(msg.Project.ID); ok {
				// Find the default repository in the list
				for _, repo := range msg.Repositories {
					if repo.ID == defaultRepoID {
						m.loading = true
						m.message = "Converting to issue in " + repo.Name + " (default)..."
						return m, convertDraft(m.apiClient, ConvertDraftMsg{
							Project:    msg.Project,
							Item:       msg.Item,
							Repository: repo,
						})
					}
				}
				// Default repo not found (maybe deleted), clear it from config
				m.config.ClearDefaultRepository(msg.Project.ID)
				if err := m.config.Save(); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to save config: %v\n", err)
				}
			}
		}
		
		// If only one repository, auto-select it and convert immediately
		if len(msg.Repositories) == 1 {
			m.loading = true
			m.message = "Converting to issue in " + msg.Repositories[0].Name + "..."
			return m, convertDraft(m.apiClient, ConvertDraftMsg{
				Project:    msg.Project,
				Item:       msg.Item,
				Repository: msg.Repositories[0],
			})
		}
		// Multiple repos - show selector
		m.repositorySelector = NewRepositorySelectorModel(msg.Repositories, msg.Project, msg.Item)
		m.repositorySelector.width = m.width
		m.repositorySelector.height = m.height
		m.currentView = viewRepositorySelector
		m.loading = false
		// Force window size update to selector
		m.repositorySelector, _ = m.repositorySelector.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		return m, m.repositorySelector.Init()

	case ConvertDraftMsg:
		// Save repository as default if requested (only if config is available)
		if msg.SaveAsDefault && m.config != nil {
			m.config.SetDefaultRepository(msg.Project.ID, msg.Repository.ID)
			if err := m.config.Save(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to save config: %v\n", err)
			}
		}
		m.loading = true
		m.message = "Converting draft to issue..."
		return m, convertDraft(m.apiClient, msg)

	case DraftConvertedMsg:
		m.loading = false
		m.message = ""
		// Reload project items to show the converted issue
		return m, loadProjectItems(m.apiClient, msg.Project)

	case ErrorMsg:
		// Extract user-friendly error message if it's an APIError
		if apiErr, ok := msg.Err.(*apierrors.APIError); ok {
			// Use the user-friendly message
			m.err = fmt.Errorf("%s", apiErr.GetUserFriendlyMessage())
			
			// Add error type indicator
			var icon string
			switch apiErr.Type {
			case apierrors.ErrorTypeRateLimit:
				icon = "⏱️ "
			case apierrors.ErrorTypePermission:
				icon = "🔒 "
			case apierrors.ErrorTypeValidation:
				icon = "⚠️ "
			case apierrors.ErrorTypeRetryable:
				icon = "🔄 "
			default:
				icon = "❌ "
			}
			m.message = icon + apiErr.GetUserFriendlyMessage()
		} else {
			m.err = msg.Err
			m.message = ""
		}
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
			case viewRepositorySelector:
				m.currentView = viewItemDetail
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
	case viewRepositorySelector:
		m.repositorySelector, cmd = m.repositorySelector.Update(msg)
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
	case viewRepositorySelector:
		return m.repositorySelector.View()
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

	warningStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFA500")).
		Padding(2, 4)

	msg := m.message
	if msg == "" {
		msg = "Initializing..."
	}

	// Use warning style if message contains warning icon
	if strings.Contains(msg, "⚠️") || strings.Contains(msg, "🔒") {
		return warningStyle.Render(msg)
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
	
	// If we have a friendly message, show it prominently
	errorText := fmt.Sprintf("Error: %v", m.err)
	if m.message != "" {
		errorText = m.message
	}
	
	if m.debugMode {
		content = lipgloss.JoinVertical(lipgloss.Left,
			errorStyle.Render(errorText),
			debugStyle.Render(fmt.Sprintf("Current view: %v\nUsername: %s\nOrgs: %v\nOwner: %s\nTechnical error: %v", 
				m.currentView, m.username, m.orgs, m.currentOwner, m.err)),
			helpStyle.Render("Press esc to continue, q to quit, ctrl+d to hide debug"),
		)
	} else {
		content = lipgloss.JoinVertical(lipgloss.Left,
			errorStyle.Render(errorText),
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

	// Load config (always succeeds, returns empty config on any error)
	cfg, _ := config.Load()

	return InitializedMsg{
		Client:   client,
		Username: username,
		Orgs:     orgs,
		Config:   cfg,
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
		// DEBUG: Log save attempt
		fmt.Fprintf(os.Stderr, "\n=== SAVE ITEM DEBUG ===\n")
		fmt.Fprintf(os.Stderr, "IsNewItem: %v\n", msg.IsNewItem)
		fmt.Fprintf(os.Stderr, "Title: %s\n", msg.Title)
		fmt.Fprintf(os.Stderr, "Assignee: %s\n", msg.Assignee)
		
		// Get assignee node ID if username provided
		var assigneeIDs []string
		if msg.Assignee != "" {
			nodeID, err := client.GetUserNodeID(msg.Assignee)
			if err != nil {
				fmt.Fprintf(os.Stderr, "ERROR: Failed to get user node ID: %v\n", err)
				return ErrorMsg{Err: fmt.Errorf("failed to get user ID for %s: %w", msg.Assignee, err)}
			}
			assigneeIDs = []string{nodeID}
			fmt.Fprintf(os.Stderr, "Assignee node ID: %s\n", nodeID)
		}

		if msg.IsNewItem {
			// Create draft issue (without assignees initially)
			fmt.Fprintf(os.Stderr, "Creating draft issue in project: %s\n", msg.Project.ID)
			item, err := client.CreateDraftIssue(models.CreateItemInput{
				ProjectID:   msg.Project.ID,
				Title:       msg.Title,
				Body:        msg.Body,
			})
			if err != nil {
				fmt.Fprintf(os.Stderr, "ERROR: CreateDraftIssue failed: %v\n", err)
				return ErrorMsg{Err: err}
			}
			
			fmt.Fprintf(os.Stderr, "Created item - ID: %s, ContentID: %s\n", item.ID, item.ContentID)
			
			// If assignees specified, update the draft issue with them
			// Use ContentID (draft issue ID), not project item ID
			if len(assigneeIDs) > 0 {
				fmt.Fprintf(os.Stderr, "Updating draft with assignees - ContentID: %s\n", item.ContentID)
				_, err = client.UpdateDraftIssue(item.ContentID, msg.Title, msg.Body, assigneeIDs)
				if err != nil {
					fmt.Fprintf(os.Stderr, "ERROR: UpdateDraftIssue failed: %v\n", err)
					// Partial success: draft created but assignee failed
					if apiErr, ok := err.(*apierrors.APIError); ok {
						return PartialSuccessMsg{
							Message:      "Draft issue created, but failed to assign user: " + apiErr.GetUserFriendlyMessage(),
							WarningError: err,
						}
					}
					return PartialSuccessMsg{
						Message:      "Draft issue created, but failed to assign user",
						WarningError: err,
					}
				}
				fmt.Fprintf(os.Stderr, "Successfully assigned user\n")
			}
		} else {
			// For updates, use ContentID (the actual draft issue/issue ID)
			fmt.Fprintf(os.Stderr, "Editing existing item - ID: %s, ContentID: %s\n", msg.Item.ID, msg.Item.ContentID)
			contentID := msg.Item.ContentID
			if contentID == "" {
				fmt.Fprintf(os.Stderr, "ERROR: ContentID is empty! Falling back to ID: %s\n", msg.Item.ID)
				// Fallback for items that might not have ContentID populated
				contentID = msg.Item.ID
			}
			fmt.Fprintf(os.Stderr, "Updating draft issue with ContentID: %s\n", contentID)
			_, err := client.UpdateDraftIssue(contentID, msg.Title, msg.Body, assigneeIDs)
			if err != nil {
				fmt.Fprintf(os.Stderr, "ERROR: UpdateDraftIssue failed: %v\n", err)
				return ErrorMsg{Err: fmt.Errorf("failed to update item: %w", err)}
			}
			fmt.Fprintf(os.Stderr, "Successfully updated item\n")
		}
		fmt.Fprintf(os.Stderr, "=== SAVE COMPLETE ===\n\n")
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

func deleteItem(client *api.Client, msg DeleteItemMsg) tea.Cmd {
	return func() tea.Msg {
		err := client.DeleteProjectItem(msg.Project.ID, msg.Item.ID)
		if err != nil {
			return ErrorMsg{Err: fmt.Errorf("failed to delete item: %w", err)}
		}
		return ItemDeletedMsg{Project: msg.Project}
	}
}

func loadRepositories(client *api.Client, owner string, isUser bool, project models.Project, item models.ProjectItem) tea.Cmd {
	return func() tea.Msg {
		repos, err := client.ListRepositories(owner, isUser)
		if err != nil {
			return ErrorMsg{Err: fmt.Errorf("failed to load repositories: %w", err)}
		}
		return RepositoriesLoadedMsg{
			Repositories: repos,
			Project:      project,
			Item:         item,
		}
	}
}

func convertDraft(client *api.Client, msg ConvertDraftMsg) tea.Cmd {
	return func() tea.Msg {
		// Get the repository node ID
		repoID := msg.Repository.ID
		
		// Convert the draft issue to a real issue
		_, err := client.ConvertDraftIssueToIssue(msg.Item.ID, repoID)
		if err != nil {
			return ErrorMsg{Err: fmt.Errorf("failed to convert draft to issue: %w", err)}
		}
		
		return DraftConvertedMsg{Project: msg.Project}
	}
}

func saveAndConvert(client *api.Client, owner string, isUser bool, msg SaveAndConvertMsg) tea.Cmd {
	return func() tea.Msg {
		// Get assignee node ID if username provided
		var assigneeIDs []string
		if msg.Assignee != "" {
			nodeID, err := client.GetUserNodeID(msg.Assignee)
			if err != nil {
				return ErrorMsg{Err: fmt.Errorf("failed to get user ID for %s: %w", msg.Assignee, err)}
			}
			assigneeIDs = []string{nodeID}
		}

		var savedItem *models.ProjectItem
		var err error

		if msg.IsNewItem {
			// Create draft issue (without assignees initially)
			savedItem, err = client.CreateDraftIssue(models.CreateItemInput{
				ProjectID:   msg.Project.ID,
				Title:       msg.Title,
				Body:        msg.Body,
			})
			if err != nil {
				return ErrorMsg{Err: fmt.Errorf("failed to create item: %w", err)}
			}
			
			// If assignees specified, update the draft issue with them
			if len(assigneeIDs) > 0 {
				_, err = client.UpdateDraftIssue(savedItem.ContentID, msg.Title, msg.Body, assigneeIDs)
				if err != nil {
					return ErrorMsg{Err: fmt.Errorf("item created but failed to assign user: %w", err)}
				}
			}
		} else {
			// For updates, use ContentID
			contentID := msg.Item.ContentID
			if contentID == "" {
				contentID = msg.Item.ID
			}
			savedItem, err = client.UpdateDraftIssue(contentID, msg.Title, msg.Body, assigneeIDs)
			if err != nil {
				return ErrorMsg{Err: fmt.Errorf("failed to update item: %w", err)}
			}
		}

		// Item saved successfully, now ready to convert
		return ItemSavedAndReadyToConvertMsg{
			Project: msg.Project,
			Item:    *savedItem,
		}
	}
}

// Messages

type InitializedMsg struct {
	Client   *api.Client
	Username string
	Orgs     []string
	Config   *config.Config
}

type ProjectsLoadedMsg struct {
	Projects []models.Project
}

type ProjectItemsLoadedMsg struct {
	Project models.Project
	Items   []models.ProjectItem
}

type ItemSavedMsg struct{}

type ItemDeletedMsg struct {
	Project models.Project
}

type ProjectCreatedMsg struct{}

type ItemSavedAndReadyToConvertMsg struct {
	Project models.Project
	Item    models.ProjectItem
}

type PartialSuccessMsg struct {
	Message      string
	WarningError error
}

type ErrorMsg struct {
	Err error
}

func searchUsersCmd(query string, owner string, isOrgProject bool) tea.Cmd {
	return func() tea.Msg {
		client, err := api.NewClient()
		if err != nil {
			return ErrorMsg{Err: fmt.Errorf("failed to create API client: %w", err)}
		}
		
		var users []string
		if isOrgProject {
			// For org projects, search only org members
			users, err = client.SearchOrgMembers(owner, query, 5)
		} else {
			// For personal projects, search all users
			users, err = client.SearchUsers(query, 5)
		}
		
		if err != nil {
			// Silently fail for user search - don't interrupt typing
			return UserSuggestionsMsg{Users: []string{}}
		}
		
		return UserSuggestionsMsg{Users: users}
	}
}
