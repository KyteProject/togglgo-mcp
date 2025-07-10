# Toggl MCP Server

A lightweight MCP (Model Context Protocol) server for Toggl time tracking, built in Go.

## Features

### Time Entry Management

- **start_time_entry** - Start a new time entry
- **stop_time_entry** - Stop the current running time entry
- **get_current_time_entry** - Get the currently running time entry
- **get_time_entries** - Get time entries with optional date filtering
- **get_time_entries_for_day** - Get time entries for a specific day (convenience)

### Project Management

- **create_project** - Create a new project
- **get_projects** - Get projects in a workspace

### Out of Scope

- **delete_project**
- **delete_entry**

## Project Structure

```shell
togglgo-mcp/
├── main.go              # Entry point
├── app/
│   ├── client.go        # Toggl API client
│   ├── handlers.go      # MCP tool handlers
│   ├── types.go         # Type definitions
│   └── utils.go         # Helper functions
├── go.mod
├── go.sum
└── README.md
```

## Setup

1. Get your Toggl API token from [Toggl Track Profile](https://track.toggl.com/profile)
2. Set the environment variable:

   ```bash
   export TOGGL_API_TOKEN=your_api_token_here
   ```

## Installation

```bash
go mod tidy
go build -o toggl-mcp
```

## Install & Usage with Claude Desktop

You can use this MCP server as a custom tool in Claude Desktop (Anthropic's desktop app) by configuring it in your Claude config file.

### 1. Build the MCP Server

```bash
go build -o toggl-mcp
```

### 3. Configure Claude Desktop

Edit (or create) your Claude Desktop config file, usually located at:

- macOS: `~/Library/Application Support/Claude/claude_desktop_config.json`
- Windows: `%APPDATA%\Claude\claude_desktop_config.json`

Add a section pointing to the binary, for example:

```json
{
  "mcpServers": {
    "toggl": {
      "command": "/Users/yourname/togglgo-mcp/toggl-mcp",
      "env": {
        "TOGGL_API_TOKEN": "your-token-here"
      }
    }
  }
}
```

**Note:**

- Make sure the `command` path points to your built `toggl-mcp` binary.
- You can set the `TOGGL_API_TOKEN` here or in your shell environment.
- Claude Desktop will launch the MCP server as needed.

### 4. Start Claude Desktop

- Restart Claude Desktop. It will detect and use your custom MCP tool.
- You can now use the Toggl tools from within Claude Desktop.

## API Reference

### Time Entry Tools

#### start_time_entry

- `description` (required) - Description of the time entry
- `workspace_id` (required) - Workspace ID
- `project_id` (optional) - Project ID

#### stop_time_entry

- `workspace_id` (required) - Workspace ID

#### get_current_time_entry

No parameters required.

#### get_time_entries

- `start_date` (optional) - Start date (YYYY-MM-DD)
- `end_date` (optional) - End date (YYYY-MM-DD)

**Important:** The Toggl API uses inclusive start, exclusive end date logic. To get entries for a single day (e.g., July 9th), use `start_date=2025-07-09` and `end_date=2025-07-10`.

#### get_time_entries_for_day

- `date` (required) - Date to get entries for (YYYY-MM-DD)

Convenience tool that automatically handles the date range for a single day.

### Project Tools

#### create_project

- `name` (required) - Project name
- `workspace_id` (required) - Workspace ID
- `color` (optional) - Project color
- `client_id` (optional) - Client ID

#### get_projects

- `workspace_id` (required) - Workspace ID
- `active` (optional) - Filter by active status

---
