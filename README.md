# timesheet

A terminal UI (TUI) application for submitting timesheet entries to Odoo directly from your workflow. It pulls your in-progress projects and their tasks from Odoo, optionally surfaces open GitHub issues for the selected project, and creates an `account.analytic.line` record with a single keystroke.

## Features

- Lists all Odoo projects with stage **In Progress**
- Loads the relevant tasks for the selected project (Odoo Support, Planning / Discovery, Development)
- Reads the project description for a `giturl:` field and fetches the open GitHub issues for that repository
- Lets you pick a date (today back to the 1st of two months ago) and a duration in 30-minute steps
- Builds the timesheet entry description from an optional GitHub issue number and a free-text area
- Animated save button with success/error feedback; resets the form for follow-up entries after saving

## Requirements

- Go 1.25 or later
- An Odoo instance accessible over HTTP/HTTPS with API key authentication
- (Optional) A `~/.gitcreds` file for authenticated GitHub API requests

## Installation

```sh
go install github.com/ThinksoftInc/timesheet@latest
```

The `timesheet` binary will be placed in `$GOPATH/bin` (or `$HOME/go/bin` if `GOPATH` is not set). Make sure that directory is on your `PATH`.

## Configuration

The application reads its connection settings from:

```
~/.config/thinksoft/config.toml
```

Create the file and the parent directory if they do not exist:

```sh
mkdir -p ~/.config/thinksoft
```

Paste the following template and fill in your details:

```toml
[connection]
hostname = "odoo.example.com"
port     = 8069
schema   = "http"
database = "mycompany"
username = "user@example.com"
apikey   = "your-api-key"
```

| Field      | Description |
|------------|-------------|
| `hostname` | Hostname of your Odoo server (no scheme, no port) |
| `port`     | TCP port — typically `8069` for HTTP or `443` for HTTPS |
| `schema`   | `http` or `https` |
| `database` | Odoo database name |
| `username` | Your Odoo login email |
| `apikey`   | An Odoo API key — generate one in *Settings → Technical → API Keys* |

### GitHub token (optional)

To avoid GitHub API rate limits and to access issues on private repositories, add your credentials to `~/.gitcreds` in the format used by `git-credential-store`:

```
https://your-github-username:your-token@github.com
```

The token only needs the `repo` scope (or `public_repo` for public repositories only).

## Usage

Run the application from any terminal:

```sh
timesheet
```

Navigate the form with **Tab** / **Shift-Tab**. Select values in dropdowns with the arrow keys and **Enter**. Press **Save** (or Tab to it and press **Enter** / **Space**) to submit the entry to Odoo. Press **Quit** or **Ctrl-C** to exit.
