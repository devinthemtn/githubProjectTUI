package api

import (
	"fmt"
	"time"

	"github.com/thomaskoefod/githubProjectTUI/internal/models"
)

// ListProjectItems retrieves items from a project
func (c *Client) ListProjectItems(projectID string, first int) ([]models.ProjectItem, error) {
	query := `query($id: ID!, $first: Int!) {
		node(id: $id) {
			... on ProjectV2 {
				items(first: $first) {
					nodes {
						id
						type
						content {
							__typename
							... on Issue {
								title
								body
								number
								state
								url
								createdAt
								updatedAt
								assignees(first: 10) {
									nodes {
										login
									}
								}
								comments(first: 50) {
									nodes {
										author {
											login
										}
										body
										createdAt
									}
								}
							}
							... on PullRequest {
								title
								body
								number
								state
								url
								createdAt
								updatedAt
								assignees(first: 10) {
									nodes {
										login
									}
								}
								comments(first: 50) {
									nodes {
										author {
											login
										}
										body
										createdAt
									}
								}
							}
							... on DraftIssue {
								title
								body
								createdAt
								updatedAt
								assignees(first: 10) {
									nodes {
										login
									}
								}
							}
						}
					}
				}
			}
		}
	}`

	variables := map[string]interface{}{
		"id":    projectID,
		"first": first,
	}

	var response struct {
		Node struct {
			Items struct {
				Nodes []struct {
					ID      string `json:"id"`
					Type    string `json:"type"`
					Content struct {
						TypeName  string    `json:"__typename"`
						Title     string    `json:"title"`
						Body      string    `json:"body"`
						Number    int       `json:"number,omitempty"`
						State     string    `json:"state,omitempty"`
						URL       string    `json:"url,omitempty"`
						CreatedAt time.Time `json:"createdAt"`
						UpdatedAt time.Time `json:"updatedAt"`
						Assignees struct {
							Nodes []struct {
								Login string `json:"login"`
							} `json:"nodes"`
						} `json:"assignees,omitempty"`
						Comments struct {
							Nodes []struct {
								Author struct {
									Login string `json:"login"`
								} `json:"author"`
								Body      string    `json:"body"`
								CreatedAt time.Time `json:"createdAt"`
							} `json:"nodes"`
						} `json:"comments,omitempty"`
					} `json:"content"`
				} `json:"nodes"`
			} `json:"items"`
		} `json:"node"`
	}

	err := c.client.Do(query, variables, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to list project items: %w", err)
	}

	items := make([]models.ProjectItem, 0)
	for _, node := range response.Node.Items.Nodes {
		assignees := make([]string, len(node.Content.Assignees.Nodes))
		for i, assignee := range node.Content.Assignees.Nodes {
			assignees[i] = assignee.Login
		}

		comments := make([]models.Comment, len(node.Content.Comments.Nodes))
		for i, comment := range node.Content.Comments.Nodes {
			comments[i] = models.Comment{
				Author:    comment.Author.Login,
				Body:      comment.Body,
				CreatedAt: comment.CreatedAt,
			}
		}

		item := models.ProjectItem{
			ID:        node.ID,
			Type:      node.Content.TypeName,
			Title:     node.Content.Title,
			Body:      node.Content.Body,
			Number:    node.Content.Number,
			State:     node.Content.State,
			URL:       node.Content.URL,
			CreatedAt: node.Content.CreatedAt,
			UpdatedAt: node.Content.UpdatedAt,
			Assignees: assignees,
			Comments:  comments,
			Fields:    make(map[string]interface{}),
		}
		items = append(items, item)
	}

	return items, nil
}

// AddProjectItem adds an item to a project
func (c *Client) AddProjectItem(input models.CreateItemInput) (*models.ProjectItem, error) {
	mutation := `mutation($input: AddProjectV2ItemByIdInput!) {
		addProjectV2ItemById(input: $input) {
			item {
				id
			}
		}
	}`

	variables := map[string]interface{}{
		"input": map[string]interface{}{
			"projectId": input.ProjectID,
			"contentId": input.ContentID,
		},
	}

	var response struct {
		AddProjectV2ItemById struct {
			Item struct {
				ID string `json:"id"`
			} `json:"item"`
		} `json:"addProjectV2ItemById"`
	}

	err := c.client.Do(mutation, variables, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to add project item: %w", err)
	}

	item := &models.ProjectItem{
		ID: response.AddProjectV2ItemById.Item.ID,
	}

	return item, nil
}

// CreateDraftIssue creates a draft issue in a project
// Note: assignees cannot be set during creation, use UpdateDraftIssue afterward
func (c *Client) CreateDraftIssue(input models.CreateItemInput) (*models.ProjectItem, error) {
	mutation := `mutation($input: AddProjectV2DraftIssueInput!) {
		addProjectV2DraftIssue(input: $input) {
			projectItem {
				id
				content {
					... on DraftIssue {
						title
						body
						createdAt
					}
				}
			}
		}
	}`

	mutationInput := map[string]interface{}{
		"projectId": input.ProjectID,
		"title":     input.Title,
		"body":      input.Body,
	}

	variables := map[string]interface{}{
		"input": mutationInput,
	}

	var response struct {
		AddProjectV2DraftIssue struct {
			ProjectItem struct {
				ID      string `json:"id"`
				Content struct {
					Title     string    `json:"title"`
					Body      string    `json:"body"`
					CreatedAt time.Time `json:"createdAt"`
				} `json:"content"`
			} `json:"projectItem"`
		} `json:"addProjectV2DraftIssue"`
	}

	err := c.client.Do(mutation, variables, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to create draft issue: %w", err)
	}

	item := &models.ProjectItem{
		ID:        response.AddProjectV2DraftIssue.ProjectItem.ID,
		Type:      "DraftIssue",
		Title:     response.AddProjectV2DraftIssue.ProjectItem.Content.Title,
		Body:      response.AddProjectV2DraftIssue.ProjectItem.Content.Body,
		CreatedAt: response.AddProjectV2DraftIssue.ProjectItem.Content.CreatedAt,
		Assignees: []string{}, // Will be empty on creation
	}

	return item, nil
}

// UpdateDraftIssue updates a draft issue
func (c *Client) UpdateDraftIssue(itemID, title, body string, assigneeIDs []string) (*models.ProjectItem, error) {
	mutation := `mutation($input: UpdateProjectV2DraftIssueInput!) {
		updateProjectV2DraftIssue(input: $input) {
			draftIssue {
				id
				title
				body
				updatedAt
				assignees(first: 10) {
					nodes {
						login
					}
				}
			}
		}
	}`

	mutationInput := map[string]interface{}{
		"draftIssueId": itemID,
	}

	if title != "" {
		mutationInput["title"] = title
	}
	if body != "" {
		mutationInput["body"] = body
	}
	if len(assigneeIDs) > 0 {
		mutationInput["assigneeIds"] = assigneeIDs
	}

	variables := map[string]interface{}{
		"input": mutationInput,
	}

	var response struct {
		UpdateProjectV2DraftIssue struct {
			DraftIssue struct {
				ID        string    `json:"id"`
				Title     string    `json:"title"`
				Body      string    `json:"body"`
				UpdatedAt time.Time `json:"updatedAt"`
				Assignees struct {
					Nodes []struct {
						Login string `json:"login"`
					} `json:"nodes"`
				} `json:"assignees"`
			} `json:"draftIssue"`
		} `json:"updateProjectV2DraftIssue"`
	}

	err := c.client.Do(mutation, variables, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to update draft issue: %w", err)
	}

	assignees := make([]string, len(response.UpdateProjectV2DraftIssue.DraftIssue.Assignees.Nodes))
	for i, node := range response.UpdateProjectV2DraftIssue.DraftIssue.Assignees.Nodes {
		assignees[i] = node.Login
	}

	item := &models.ProjectItem{
		ID:        response.UpdateProjectV2DraftIssue.DraftIssue.ID,
		Type:      "DraftIssue",
		Title:     response.UpdateProjectV2DraftIssue.DraftIssue.Title,
		Body:      response.UpdateProjectV2DraftIssue.DraftIssue.Body,
		UpdatedAt: response.UpdateProjectV2DraftIssue.DraftIssue.UpdatedAt,
		Assignees: assignees,
	}

	return item, nil
}

// DeleteProjectItem removes an item from a project
func (c *Client) DeleteProjectItem(projectID, itemID string) error {
	mutation := `mutation($input: DeleteProjectV2ItemInput!) {
		deleteProjectV2Item(input: $input) {
			deletedItemId
		}
	}`

	variables := map[string]interface{}{
		"input": map[string]interface{}{
			"projectId": projectID,
			"itemId":    itemID,
		},
	}

	var response map[string]interface{}

	err := c.client.Do(mutation, variables, &response)
	if err != nil {
		return fmt.Errorf("failed to delete project item: %w", err)
	}

	return nil
}
