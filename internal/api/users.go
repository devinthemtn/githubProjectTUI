package api

import (
	"fmt"
	"strings"
)

// SearchUsers searches for GitHub users by username
func (c *Client) SearchUsers(query string, limit int) ([]string, error) {
	if query == "" {
		return []string{}, nil
	}

	// Use GraphQL search to find users
	gqlQuery := `query($query: String!) {
		search(query: $query, type: USER, first: 10) {
			nodes {
				... on User {
					login
				}
			}
		}
	}`

	// Build search query - search for users whose login starts with or contains the query
	searchQuery := fmt.Sprintf("%s in:login type:user", query)

	variables := map[string]interface{}{
		"query": searchQuery,
	}

	var response struct {
		Search struct {
			Nodes []struct {
				Login string `json:"login"`
			} `json:"nodes"`
		} `json:"search"`
	}

	err := c.client.Do(gqlQuery, variables, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to search users: %w", err)
	}

	users := make([]string, 0, len(response.Search.Nodes))
	for _, node := range response.Search.Nodes {
		// Filter to only include users whose login starts with query (case-insensitive)
		if strings.HasPrefix(strings.ToLower(node.Login), strings.ToLower(query)) {
			users = append(users, node.Login)
			if len(users) >= limit {
				break
			}
		}
	}

	// If we didn't get enough prefix matches, add contains matches
	if len(users) < limit {
		for _, node := range response.Search.Nodes {
			if !strings.HasPrefix(strings.ToLower(node.Login), strings.ToLower(query)) &&
				strings.Contains(strings.ToLower(node.Login), strings.ToLower(query)) {
				users = append(users, node.Login)
				if len(users) >= limit {
					break
				}
			}
		}
	}

	return users, nil
}
