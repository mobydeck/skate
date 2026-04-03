# 🛹 Skate

Access Mattermost Boards tasks from CLI and AI agents via MCP.

Skate lets you manage project tasks, track time, and collaborate with AI coding agents — all from your terminal. It's a thin, stateless Go client over the Mattermost Boards API with zero runtime dependencies.

## Install

Download a binary from [Releases](https://github.com/mobydeck/skate/releases), or build from source:

```bash
git clone https://github.com/mobydeck/skate
cd skate
make install   # builds and installs to ~/.local/bin/
```

Cross-build for all platforms:

```bash
just cross-build   # outputs to dist/
```

## Setup

### 1. Initialize global config

```bash
skate init
```

You'll need:
- Your Mattermost server URL (e.g., `https://mm.example.com`)
- A [personal access token](https://docs.mattermost.com/developer/personal-access-tokens.html)
- Your team ID (auto-detected if only one team)

Re-run `skate init` anytime to update settings — existing values are shown as defaults.

### 2. Initialize per-project config

In your project directory:

```bash
skate local-init
```

This creates `.skate.yaml` with the board ID for this project. Skate walks up directories to find the nearest `.skate.yaml` and merges it with the global config, so nested folders inherit settings. Re-run to change the board.

### 3. Connect an AI agent

```bash
skate setup claude-code   # Claude Code
skate setup cursor        # Cursor
skate setup codex         # Codex
skate setup opencode      # OpenCode
skate setup roocode       # RooCode
```

This registers the MCP server and installs a skill file that teaches the agent how to use Skate.

Use `--project` / `-p` flag to install for the current project only (writes to `.mcp.json` instead of global config).

### 4. Bootstrap an AI agent session

If an agent doesn't auto-discover MCP or skills, paste the output of `skate prompt` into the chat:

```bash
skate prompt claude-code   # or: cursor, codex
```

```
Before starting work, read and follow rules in /home/user/.claude/skills/skate/SKILL.md
```

## CLI Commands

### Boards

```
$ skate boards
ID                           TITLE                   TYPE
bto6yyshu77yj5x5mbu34cnhzcr  🗓️ Sprint Planner       Private
bht3p6dq1i381jrzx8kngmt4gjw  🎯 Arthur – Tasks       Private
bqn9n36errtymuqk1ph7xy7coiw  📖 Read Books           Private
```

```bash
skate boards                          # List all boards
skate boards --json                   # JSON output
skate boards --yaml                   # YAML output
```

### Tasks

By default, `skate tasks` shows only **Not Started** and **In Progress** tasks, sorted by priority (high to low).

```
$ skate tasks
ID                           TITLE                        STATUS       PRIORITY   ASSIGNEE
c4cf6f4wzbjgxdm3hpa7iygtjdo  Task translation middleware  Not Started  2. Medium
cuppcm819atnixx71qg9i485jsr  listing tasks                In Progress  1. High 🔥
```

```bash
skate tasks                           # Active tasks from .skate.yaml board
skate tasks --all                     # All tasks regardless of status
skate tasks --status "Not Started"    # Filter by specific status
skate tasks --board <BOARD_ID>        # Specific board
skate tasks --mine                    # Only tasks assigned to you
skate tasks --all-users               # All users (overrides only_mine config)
skate tasks --json                    # JSON output
skate task <TASK_ID>                  # View full task details (markdown)
skate task <TASK_ID> --json           # Full task data as JSON
skate task-files <TASK_ID>            # List attached files
skate download <BOARD_ID> <FILE_ID>   # Download a file
```

Task detail renders as markdown:

```
$ skate task c4cf6f4wzbjgxdm3hpa7iygtjdo

# 📪 Task translation middleware

| Property | Value |
|----------|-------|
| Status   | In Progress |
| Priority | 2. Medium |

## Description
Add translation middleware for non-English board content...

## Comments
**@arthur** (Apr 3, 2026):
> Implemented translation middleware with OpenAI SDK

## Time Tracking
- @arthur: 00:08

Total: 00:08
```

### Task Management

```bash
skate create "Fix login bug" --status "Not Started" --priority "High"
skate create "New feature" --description "Detailed description here"
skate update-status <TASK_ID> "In Progress"
skate comment <TASK_ID> "Implemented the fix, running tests"
skate attach <TASK_ID> ./screenshot.png
```

### Time Tracking

Time tracking requires the Mattermost Boards time tracking plugin. If unavailable, timer commands will print a message and continue without error.

```
$ skate timer-start c4cf6f4wzbjgxdm3hpa7iygtjdo
Timer started on: Task translation middleware

$ skate timer-stop --notes "Completed implementation"
Timer stopped: Task translation middleware — 00:08
```

```bash
skate timer-start <TASK_ID>                     # Start timer (auto-stops previous)
skate timer-stop --notes "Completed feature"    # Stop running timer
skate time-add <TASK_ID> 01:30 --notes "Code review"   # Add manual time
skate time-add <TASK_ID> 02:00 --date 2026-04-01       # Backdate entry
```

### Output Formats

All data commands support `--json` / `-j` and `--yaml` / `-y` flags:

```bash
skate boards --json
skate tasks --yaml
skate task <ID> -j
skate task-files <ID> -y
```

## MCP Tools

When connected via MCP, AI agents can use these tools:

| Tool | Description |
|------|-------------|
| `skate_boards` | List available boards |
| `skate_tasks` | List tasks (default: active only, use `show_all` for all) |
| `skate_task` | Get full task details as markdown |
| `skate_update_status` | Change task status |
| `skate_create_task` | Create a new task |
| `skate_comment` | Add a comment to a task |
| `skate_timer_start` | Start timer on a task |
| `skate_timer_stop` | Stop running timer with notes |
| `skate_time_add` | Add manual time entry |

## Example Prompts for AI Agents

```
"Look at the board tasks, pick the highest priority unstarted task, and implement it"

"Check my current tasks and update the one I'm working on"

"Create a task for the bug I just found in the auth module"

"Start a timer, implement the task, then stop the timer with notes"

"List all tasks assigned to me that are In Progress"

"Take the next task by priority"

"Work on next task"
```

## Translation

Skate can automatically translate non-English board content to English using any OpenAI-compatible API (OpenAI, Ollama, OpenRouter, etc.).

Enable in config:

```yaml
# ~/.config/skate.yaml
translate:
  enabled: true
  provider: openai         # or ollama, openrouter
  model: gpt-5-mini        # any chat model
  base_url: ""             # custom endpoint (e.g., http://localhost:11434/v1 for Ollama)
  api_key: "sk-..."        # API key (not needed for Ollama)
```

Translation uses a fast heuristic to detect non-English text and only calls the API when needed. English content passes through untouched.

## Config Files

- **Global:** `~/.config/skate.yaml` (Linux), `%AppData%\skate\skate.yaml` (Windows)
- **Local:** `.skate.yaml` (per project, walks up directories)
- **User cache:** `~/.cache/skate/users.yaml` (Linux), `%LocalAppData%\skate\users.yaml` (Windows)

```yaml
# ~/.config/skate.yaml
mattermost_url: "https://mm.example.com"
token: "your-personal-access-token"
team_id: "your-team-id"
only_mine: false           # show only your tasks by default

translate:
  enabled: false
  provider: openai
  model: gpt-5-mini
  base_url: ""
  api_key: ""
```

```yaml
# .skate.yaml (in project root)
board_id: "your-board-id"
```

## Environment Variables

All config values can be overridden with environment variables:

| Variable | Description |
|----------|-------------|
| `SKATE_URL` | Mattermost server URL |
| `SKATE_TOKEN` | Personal access token |
| `SKATE_TEAM_ID` | Team ID |
| `SKATE_BOARD_ID` | Default board ID |
| `SKATE_TRANSLATE_ENABLED` | Enable translation (`true`/`1`) |
| `SKATE_TRANSLATE_PROVIDER` | Translation provider |
| `SKATE_TRANSLATE_MODEL` | Model name |
| `SKATE_TRANSLATE_BASE_URL` | Custom API endpoint |
| `SKATE_TRANSLATE_API_KEY` | API key |

## Building

```bash
make build         # build for current platform
make install       # build + install to ~/.local/bin/
make test          # run tests
just cross-build   # cross-build for linux/darwin/windows (amd64 + arm64)
just release       # create draft GitHub release with all binaries
```

Version is set at build time via ldflags. It defaults to `dev`, or auto-derives from git tags:

```bash
VERSION=1.0.0 make build   # explicit version
```

## Architecture

- **Pure Go** — single static binary, no runtime dependencies, cross-compilable to 5 platforms
- **Stateless** — no local database, all data lives in Mattermost
- **User cache** — resolved usernames cached in `~/.cache/skate/users.yaml` (Linux) or `%LocalAppData%\skate\` (Windows)
- **MCP stdio** — agents start skate on demand, zero idle cost
- **Config merging** — global config + local `.skate.yaml` + env vars (env wins)
- **Version** — set via ldflags, shared across CLI, HTTP client User-Agent, and MCP server

## License

See [LICENSE](LICENSE) for details.
