# GitHub Projects TUI

A Terminal User Interface (TUI) for managing GitHub Projects V2, built with Go and the Charm ecosystem.

## Features

- 🎨 Beautiful terminal interface powered by Bubble Tea
- 🔐 Seamless authentication using GitHub CLI
- 📋 View, create, and edit GitHub Projects V2
- ⚡ Fast and intuitive keyboard-driven navigation
- 🎯 Manage project items (issues, PRs, draft items)

## Prerequisites

- Go 1.21 or later
- [GitHub CLI](https://cli.github.com/) (`gh`) installed and authenticated
  ```bash
  gh auth login
  ```

## Installation

### From Source

```bash
git clone https://github.com/thomaskoefod/githubProjectTUI.git
cd githubProjectTUI
go build -o ghptui ./cmd/ghptui
```

### Using Go Install

```bash
go install github.com/thomaskoefod/githubProjectTUI/cmd/ghptui@latest
```

## Usage

Run the application:

```bash
./ghptui
```

Or if installed via `go install`:

```bash
ghptui
```

### Keyboard Shortcuts

- **Navigation**: `j`/`k` or `↓`/`↑` to move up/down
- **Select**: `Enter` to open/select
- **Back**: `Esc` to go back
- **New**: `n` to create new project
- **Edit**: `e` to edit selected item
- **Delete**: `d` to delete selected item
- **Help**: `?` to toggle help screen
- **Quit**: `q` or `Ctrl+C` to exit

## Development Status

This project is currently in active development.

### Completed Features

- ✅ Phase 1: Project setup and basic Bubble Tea initialization
- ✅ Phase 2: GitHub API integration (authentication, projects, items)
- ✅ Phase 3: Core UI components (project list, detail view, item editor)
- ✅ Phase 4: Navigation and state management

### Current Status

The application is now functional! You can:
- 🎯 View your GitHub Projects V2
- 📋 Browse project items in a table view
- ✏️ Create and edit draft issues
- 🔍 Filter and search projects
- ⌨️ Navigate with intuitive keyboard shortcuts

### Coming Soon

- 🚧 Project creation UI
- 🚧 Link existing issues/PRs to projects
- 🚧 Delete confirmations
- 🚧 Better error handling and retries

## Authentication

The application uses your existing GitHub CLI authentication. Make sure you have the following scopes:

- `project` - For accessing GitHub Projects
- `repo` - For repository access
- `read:org` - For organization projects

You can verify your authentication with:

```bash
gh auth status
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License - see LICENSE file for details

## Built With

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) - Styling
- [Bubbles](https://github.com/charmbracelet/bubbles) - TUI components
- [go-gh](https://github.com/cli/go-gh) - GitHub API client

## Acknowledgments

- The amazing [Charm](https://charm.sh/) team for their excellent TUI libraries
- GitHub for the Projects V2 API
