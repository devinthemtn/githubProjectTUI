package api

import (
	"fmt"
	"strings"
)

// GetOrgMembers retrieves all members of an organization
func (c *Client) GetOrgMembers(org string, limit int) ([]string, error) {
	query := `query($org: String!, $first: Int!) {
		organization(login: $org) {
			membersWithRole(first: $first) {
				nodes {
					login
				}
			}
		}
	}`

	variables := map[string]interface{}{
		"org":   org,
		"first": limit,
	}

	var response struct {
		Organization struct {
			MembersWithRole struct {
				Nodes []struct {
					Login string `json:"login"`
				} `json:"nodes"`
			} `json:"membersWithRole"`
		} `json:"organization"`
	}

	err := c.client.Do(query, variables, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get org members: %w", err)
	}

	members := make([]string, len(response.Organization.MembersWithRole.Nodes))
	for i, node := range response.Organization.MembersWithRole.Nodes {
		members[i] = node.Login
	}

	return members, nil
}

// SearchOrgMembers searches for organization members by username
func (c *Client) SearchOrgMembers(org string, query string, limit int) ([]string, error) {
	if query == "" {
		return []string{}, nil
	}

	// Get all org members (up to 100)
	members, err := c.GetOrgMembers(org, 100)
	if err != nil {
		return nil, fmt.Errorf("failed to get org members: %w", err)
	}

	// Filter members by query
	matches := make([]string, 0, limit)
	queryLower := strings.ToLower(query)

	// First pass: prefix matches
	for _, member := range members {
		if strings.HasPrefix(strings.ToLower(member), queryLower) {
			matches = append(matches, member)
			if len(matches) >= limit {
				return matches, nil
			}
		}
	}

	// Second pass: contains matches
	for _, member := range members {
		if !strings.HasPrefix(strings.ToLower(member), queryLower) &&
			strings.Contains(strings.ToLower(member), queryLower) {
			matches = append(matches, member)
			if len(matches) >= limit {
				return matches, nil
			}
		}
	}

	return matches, nil
}
