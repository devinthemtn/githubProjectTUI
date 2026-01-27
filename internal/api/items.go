package api

import (
	"fmt"
	"os"
	"time"

	apierrors "github.com/thomaskoefod/githubProjectTUI/internal/errors"
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
								id
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
								id
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
								id
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
						ID        string    `json:"id"`
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
			ContentID: node.Content.ID,
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

// CreateDraftIssue creates a draft issue in a project with retry logic
// Note: assignees cannot be set during creation, use UpdateDraftIssue afterward
func (c *Client) CreateDraftIssue(input models.CreateItemInput) (*models.ProjectItem, error) {
	fmt.Fprintf(os.Stderr, "\n>>> CreateDraftIssue called\n")
	fmt.Fprintf(os.Stderr, "ProjectID: %s\n", input.ProjectID)
	fmt.Fprintf(os.Stderr, "Title: %s\n", input.Title)
	
	var result *models.ProjectItem
	
	// Retry wrapper
	err := apierrors.Retry(func() error {
		mutation := `mutation($input: AddProjectV2DraftIssueInput!) {
			addProjectV2DraftIssue(input: $input) {
				projectItem {
					id
					content {
						... on DraftIssue {
							id
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
						ID        string    `json:"id"`
						Title     string    `json:"title"`
						Body      string    `json:"body"`
						CreatedAt time.Time `json:"createdAt"`
					} `json:"content"`
				} `json:"projectItem"`
			} `json:"addProjectV2DraftIssue"`
		}

		if err := c.client.Do(mutation, variables, &response); err != nil {
			fmt.Fprintf(os.Stderr, "CreateDraftIssue GraphQL error: %v\n", err)
			// Classify the error
			classified := apierrors.ClassifyError(err, 0)
			return classified
		}

		result = &models.ProjectItem{
			ID:        response.AddProjectV2DraftIssue.ProjectItem.ID,
			ContentID: response.AddProjectV2DraftIssue.ProjectItem.Content.ID,
			Type:      "DraftIssue",
			Title:     response.AddProjectV2DraftIssue.ProjectItem.Content.Title,
			Body:      response.AddProjectV2DraftIssue.ProjectItem.Content.Body,
			CreatedAt: response.AddProjectV2DraftIssue.ProjectItem.Content.CreatedAt,
			Assignees: []string{}, // Will be empty on creation
		}

		fmt.Fprintf(os.Stderr, "CreateDraftIssue success - ProjectItem.ID: %s, Content.ID: %s\n", result.ID, result.ContentID)
		return nil
	}, apierrors.DefaultRetryConfig())

	if err != nil {
		return nil, err
	}

	return result, nil
}

// UpdateDraftIssue updates a draft issue with retry logic
func (c *Client) UpdateDraftIssue(itemID, title, body string, assigneeIDs []string) (*models.ProjectItem, error) {
	fmt.Fprintf(os.Stderr, "\n>>> UpdateDraftIssue called\n")
	fmt.Fprintf(os.Stderr, "DraftIssueID: %s\n", itemID)
	fmt.Fprintf(os.Stderr, "Title: %s\n", title)
	fmt.Fprintf(os.Stderr, "Body length: %d\n", len(body))
	fmt.Fprintf(os.Stderr, "AssigneeIDs: %v\n", assigneeIDs)
	
	var result *models.ProjectItem
	
	err := apierrors.Retry(func() error {
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
		
		fmt.Fprintf(os.Stderr, "Mutation input: %+v\n", mutationInput)

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

		if err := c.client.Do(mutation, variables, &response); err != nil {
			fmt.Fprintf(os.Stderr, "UpdateDraftIssue GraphQL error: %v\n", err)
			return apierrors.ClassifyError(err, 0)
		}

		assignees := make([]string, len(response.UpdateProjectV2DraftIssue.DraftIssue.Assignees.Nodes))
		for i, node := range response.UpdateProjectV2DraftIssue.DraftIssue.Assignees.Nodes {
			assignees[i] = node.Login
		}

		result = &models.ProjectItem{
			ID:        response.UpdateProjectV2DraftIssue.DraftIssue.ID,
			Type:      "DraftIssue",
			Title:     response.UpdateProjectV2DraftIssue.DraftIssue.Title,
			Body:      response.UpdateProjectV2DraftIssue.DraftIssue.Body,
			UpdatedAt: response.UpdateProjectV2DraftIssue.DraftIssue.UpdatedAt,
			Assignees: assignees,
		}

		fmt.Fprintf(os.Stderr, "UpdateDraftIssue success - ID: %s, Assignees: %v\n", result.ID, result.Assignees)
		return nil
	}, apierrors.DefaultRetryConfig())

	if err != nil {
		return nil, err
	}

	return result, nil
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

// ConvertDraftIssueToIssue converts a draft issue to a real GitHub issue with retry logic
func (c *Client) ConvertDraftIssueToIssue(projectItemID, repositoryID string) (*models.ProjectItem, error) {
	fmt.Fprintf(os.Stderr, "\n>>> ConvertDraftIssueToIssue called\n")
	fmt.Fprintf(os.Stderr, "ProjectItemID: %s\n", projectItemID)
	fmt.Fprintf(os.Stderr, "RepositoryID: %s\n", repositoryID)
	
	var result *models.ProjectItem
	
	err := apierrors.Retry(func() error {
		mutation := `mutation($input: ConvertProjectV2DraftIssueItemToIssueInput!) {
			convertProjectV2DraftIssueItemToIssue(input: $input) {
				projectV2Item {
					id
				}
				newIssue {
					id
					number
					title
					url
				}
			}
		}`

		variables := map[string]interface{}{
			"input": map[string]interface{}{
				"projectV2ItemId": projectItemID,
				"repositoryId":    repositoryID,
			},
		}

		var response struct {
			ConvertProjectV2DraftIssueItemToIssue struct {
				ProjectV2Item struct {
					ID string `json:"id"`
				} `json:"projectV2Item"`
				NewIssue struct {
					ID     string `json:"id"`
					Number int    `json:"number"`
					Title  string `json:"title"`
					URL    string `json:"url"`
				} `json:"newIssue"`
			} `json:"convertProjectV2DraftIssueItemToIssue"`
		}

		if err := c.client.Do(mutation, variables, &response); err != nil {
			fmt.Fprintf(os.Stderr, "ConvertDraftIssueToIssue GraphQL error: %v\n", err)
			fmt.Fprintf(os.Stderr, "Variables sent: %+v\n", variables)
			return apierrors.ClassifyError(err, 0)
		}

		result = &models.ProjectItem{
			ID:     response.ConvertProjectV2DraftIssueItemToIssue.ProjectV2Item.ID,
			Type:   "Issue",
			Title:  response.ConvertProjectV2DraftIssueItemToIssue.NewIssue.Title,
			Number: response.ConvertProjectV2DraftIssueItemToIssue.NewIssue.Number,
			URL:    response.ConvertProjectV2DraftIssueItemToIssue.NewIssue.URL,
		}

		fmt.Fprintf(os.Stderr, "ConvertDraftIssueToIssue success - Issue #%d: %s\n", result.Number, result.URL)
		return nil
	}, apierrors.DefaultRetryConfig())

	if err != nil {
		return nil, err
	}

	return result, nil
}
