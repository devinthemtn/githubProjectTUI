package api

import (
"fmt"
"time"

"github.com/cli/go-gh/v2/pkg/api"
"github.com/thomaskoefod/githubProjectTUI/internal/models"
)

// Client wraps the GitHub API client for Projects V2
type Client struct {
client *api.GraphQLClient
}

// NewClient creates a new API client
func NewClient() (*Client, error) {
opts := api.ClientOptions{}
client, err := api.NewGraphQLClient(opts)
if err != nil {
return nil, fmt.Errorf("failed to create API client: %w", err)
}

return &Client{
client: client,
}, nil
}

// GetViewer returns information about the authenticated user
func (c *Client) GetViewer() (string, error) {
query := `query {
viewer {
login
}
}`

var response struct {
Viewer struct {
Login string
}
}

err := c.client.Do(query, nil, &response)
if err != nil {
return "", fmt.Errorf("failed to get viewer: %w", err)
}

return response.Viewer.Login, nil
}

// ListUserProjects retrieves all projects for the authenticated user
func (c *Client) ListUserProjects(login string, first int) ([]models.Project, error) {
query := `query($login: String!, $first: Int!) {
user(login: $login) {
projectsV2(first: $first) {
nodes {
id
number
title
shortDescription
public
closed
url
createdAt
updatedAt
items {
totalCount
}
}
}
}
}`

variables := map[string]interface{}{
"login": login,
"first": first,
}

var response struct {
User struct {
ProjectsV2 struct {
Nodes []struct {
ID               string
Number           int
Title            string
ShortDescription string
Public           bool
Closed           bool
URL              string
CreatedAt        time.Time
UpdatedAt        time.Time
Items            struct {
TotalCount int
}
}
}
}
}

err := c.client.Do(query, variables, &response)
if err != nil {
return nil, fmt.Errorf("failed to list user projects: %w", err)
}

projects := make([]models.Project, len(response.User.ProjectsV2.Nodes))
for i, node := range response.User.ProjectsV2.Nodes {
projects[i] = models.Project{
ID:               node.ID,
Number:           node.Number,
Title:            node.Title,
ShortDescription: node.ShortDescription,
Public:           node.Public,
Closed:           node.Closed,
URL:              node.URL,
CreatedAt:        node.CreatedAt,
UpdatedAt:        node.UpdatedAt,
ItemCount:        node.Items.TotalCount,
Owner: models.ProjectOwner{
Login: login,
Type:  "User",
},
}
}

return projects, nil
}

// ListOrgProjects retrieves all projects for an organization
func (c *Client) ListOrgProjects(org string, first int) ([]models.Project, error) {
query := `query($org: String!, $first: Int!) {
organization(login: $org) {
projectsV2(first: $first) {
nodes {
id
number
title
shortDescription
public
closed
url
createdAt
updatedAt
items {
totalCount
}
}
}
}
}`

variables := map[string]interface{}{
"org":   org,
"first": first,
}

var response struct {
Organization struct {
ProjectsV2 struct {
Nodes []struct {
ID               string
Number           int
Title            string
ShortDescription string
Public           bool
Closed           bool
URL              string
CreatedAt        time.Time
UpdatedAt        time.Time
Items            struct {
TotalCount int
}
}
}
}
}

err := c.client.Do(query, variables, &response)
if err != nil {
return nil, fmt.Errorf("failed to list org projects: %w", err)
}

projects := make([]models.Project, len(response.Organization.ProjectsV2.Nodes))
for i, node := range response.Organization.ProjectsV2.Nodes {
projects[i] = models.Project{
ID:               node.ID,
Number:           node.Number,
Title:            node.Title,
ShortDescription: node.ShortDescription,
Public:           node.Public,
Closed:           node.Closed,
URL:              node.URL,
CreatedAt:        node.CreatedAt,
UpdatedAt:        node.UpdatedAt,
ItemCount:        node.Items.TotalCount,
Owner: models.ProjectOwner{
Login: org,
Type:  "Organization",
},
}
}

return projects, nil
}

// GetUserOrganizations retrieves the user's organizations
func (c *Client) GetUserOrganizations(username string) ([]string, error) {
query := `query($login: String!) {
user(login: $login) {
organizations(first: 100) {
nodes {
login
}
}
}
}`

variables := map[string]interface{}{
"login": username,
}

var response struct {
User struct {
Organizations struct {
Nodes []struct {
Login string
}
}
}
}

err := c.client.Do(query, variables, &response)
if err != nil {
return nil, fmt.Errorf("failed to get organizations: %w", err)
}

orgs := make([]string, len(response.User.Organizations.Nodes))
for i, node := range response.User.Organizations.Nodes {
orgs[i] = node.Login
}

return orgs, nil
}

// GetUserNodeID retrieves the node ID for a user
func (c *Client) GetUserNodeID(username string) (string, error) {
query := `query($login: String!) {
user(login: $login) {
id
}
}`

variables := map[string]interface{}{
"login": username,
}

var response struct {
User struct {
ID string
}
}

err := c.client.Do(query, variables, &response)
if err != nil {
return "", fmt.Errorf("failed to get user node ID: %w", err)
}

return response.User.ID, nil
}

// GetOrgNodeID retrieves the node ID for an organization
func (c *Client) GetOrgNodeID(org string) (string, error) {
query := `query($login: String!) {
organization(login: $login) {
id
}
}`

variables := map[string]interface{}{
"login": org,
}

var response struct {
Organization struct {
ID string
}
}

err := c.client.Do(query, variables, &response)
if err != nil {
return "", fmt.Errorf("failed to get org node ID: %w", err)
}

return response.Organization.ID, nil
}

// CreateProject creates a new project
func (c *Client) CreateProject(input models.CreateProjectInput) (*models.Project, error) {
mutation := `mutation($input: CreateProjectV2Input!) {
createProjectV2(input: $input) {
projectV2 {
id
number
title
shortDescription
public
url
createdAt
}
}
}`

mutationInput := map[string]interface{}{
"ownerId": input.OwnerID,
"title":   input.Title,
}

if input.ShortDescription != "" {
mutationInput["shortDescription"] = input.ShortDescription
}

variables := map[string]interface{}{
"input": mutationInput,
}

var response struct {
CreateProjectV2 struct {
ProjectV2 struct {
ID               string
Number           int
Title            string
ShortDescription string
Public           bool
URL              string
CreatedAt        time.Time
}
}
}

err := c.client.Do(mutation, variables, &response)
if err != nil {
return nil, fmt.Errorf("failed to create project: %w", err)
}

project := &models.Project{
ID:               response.CreateProjectV2.ProjectV2.ID,
Number:           response.CreateProjectV2.ProjectV2.Number,
Title:            response.CreateProjectV2.ProjectV2.Title,
ShortDescription: response.CreateProjectV2.ProjectV2.ShortDescription,
Public:           response.CreateProjectV2.ProjectV2.Public,
URL:              response.CreateProjectV2.ProjectV2.URL,
CreatedAt:        response.CreateProjectV2.ProjectV2.CreatedAt,
}

return project, nil
}
