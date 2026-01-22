package api

import (
	"fmt"

	"github.com/thomaskoefod/githubProjectTUI/internal/models"
)

// GetRepositoryNodeID retrieves the node ID for a repository
func (c *Client) GetRepositoryNodeID(owner, name string) (string, error) {
	query := `query($owner: String!, $name: String!) {
		repository(owner: $owner, name: $name) {
			id
		}
	}`

	variables := map[string]interface{}{
		"owner": owner,
		"name":  name,
	}

	var response struct {
		Repository struct {
			ID string `json:"id"`
		} `json:"repository"`
	}

	err := c.client.Do(query, variables, &response)
	if err != nil {
		return "", fmt.Errorf("failed to get repository node ID: %w", err)
	}

	return response.Repository.ID, nil
}

// ListRepositories retrieves repositories accessible to the user or organization
func (c *Client) ListRepositories(owner string, isUser bool) ([]models.Repository, error) {
	var query string
	
	if isUser {
		query = `query($login: String!) {
			user(login: $login) {
				repositories(first: 100, orderBy: {field: UPDATED_AT, direction: DESC}) {
					nodes {
						id
						name
						owner {
							login
						}
						description
						isPrivate
					}
				}
			}
		}`
	} else {
		query = `query($login: String!) {
			organization(login: $login) {
				repositories(first: 100, orderBy: {field: UPDATED_AT, direction: DESC}) {
					nodes {
						id
						name
						owner {
							login
						}
						description
						isPrivate
					}
				}
			}
		}`
	}

	variables := map[string]interface{}{
		"login": owner,
	}

	var response struct {
		User struct {
			Repositories struct {
				Nodes []struct {
					ID          string `json:"id"`
					Name        string `json:"name"`
					Owner       struct {
						Login string `json:"login"`
					} `json:"owner"`
					Description string `json:"description"`
					IsPrivate   bool   `json:"isPrivate"`
				} `json:"nodes"`
			} `json:"repositories"`
		} `json:"user,omitempty"`
		Organization struct {
			Repositories struct {
				Nodes []struct {
					ID          string `json:"id"`
					Name        string `json:"name"`
					Owner       struct {
						Login string `json:"login"`
					} `json:"owner"`
					Description string `json:"description"`
					IsPrivate   bool   `json:"isPrivate"`
				} `json:"nodes"`
			} `json:"repositories"`
		} `json:"organization,omitempty"`
	}

	err := c.client.Do(query, variables, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to list repositories: %w", err)
	}

	var repos []models.Repository
	var nodes []struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Owner       struct {
			Login string `json:"login"`
		} `json:"owner"`
		Description string `json:"description"`
		IsPrivate   bool   `json:"isPrivate"`
	}

	if isUser {
		nodes = response.User.Repositories.Nodes
	} else {
		nodes = response.Organization.Repositories.Nodes
	}

	for _, node := range nodes {
		repos = append(repos, models.Repository{
			ID:          node.ID,
			Name:        node.Name,
			Owner:       node.Owner.Login,
			Description:   node.Description,
			IsPrivate:   node.IsPrivate,
		})
	}

	return repos, nil
}
