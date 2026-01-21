package models

import "time"

// Project represents a GitHub Project V2
type Project struct {
	ID          string
	Number      int
	Title       string
	ShortDescription string
	Public      bool
	Closed      bool
	URL         string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Owner       ProjectOwner
	ItemCount   int
}

// ProjectOwner represents the owner of a project (user or organization)
type ProjectOwner struct {
	Login string
	Type  string // "User" or "Organization"
}

// ProjectItem represents an item in a project
type ProjectItem struct {
	ID        string
	Type      string // "ISSUE", "PULL_REQUEST", "DRAFT_ISSUE"
	Title     string
	Body      string
	Number    int    // For issues/PRs
	State     string // For issues/PRs
	URL       string
	CreatedAt time.Time
	UpdatedAt time.Time
	Fields    map[string]interface{}
}

// ProjectField represents a custom field in a project
type ProjectField struct {
	ID       string
	Name     string
	DataType string // "TEXT", "NUMBER", "DATE", "SINGLE_SELECT", "ITERATION"
	Options  []ProjectFieldOption
}

// ProjectFieldOption represents an option for single-select fields
type ProjectFieldOption struct {
	ID    string
	Name  string
	Color string
}

// CreateProjectInput represents input for creating a new project
type CreateProjectInput struct {
	OwnerID          string
	Title            string
	ShortDescription string
	Public           bool
}

// UpdateProjectInput represents input for updating a project
type UpdateProjectInput struct {
	ProjectID        string
	Title            *string
	ShortDescription *string
	Public           *bool
	Closed           *bool
}

// CreateItemInput represents input for creating a new project item
type CreateItemInput struct {
	ProjectID   string
	ContentID   string // Optional: ID of issue/PR to add
	Title       string // For draft issues
	Body        string // For draft issues
}

// UpdateItemInput represents input for updating a project item
type UpdateItemInput struct {
	ProjectID string
	ItemID    string
	FieldID   string
	Value     interface{}
}
