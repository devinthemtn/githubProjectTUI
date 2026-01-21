package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/thomaskoefod/githubProjectTUI/internal/api"
	"github.com/thomaskoefod/githubProjectTUI/internal/ui"
)

func main() {
	client, err := api.NewClient()
	if err != nil {
		fmt.Printf("Failed to create client: %v\n", err)
		os.Exit(1)
	}

	viewer, _ := client.GetViewer()
	orgs, _ := client.GetUserOrganizations(viewer)

	fmt.Printf("Username: %s\n", viewer)
	fmt.Printf("Orgs: %v\n", orgs)

	if len(orgs) > 0 {
		projects, err := client.ListOrgProjects(orgs[0], 10)
		if err != nil {
			fmt.Printf("Error loading projects: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("\nLoaded %d projects\n", len(projects))
		
		// Test creating the UI model
		projectList := ui.NewProjectListModel(projects)
		fmt.Printf("Created project list model with %d items\n", len(projects))
		
		// Send window size update
		projectList, _ = projectList.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
		
		// Try to render it
		view := projectList.View()
		fmt.Printf("\nRendered view length: %d characters\n", len(view))
		fmt.Printf("\nView output:\n%s\n", view)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
