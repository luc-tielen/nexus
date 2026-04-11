# Nexus

A personal AI assistant, powered by Claude Code.

## Installation

```sh
nexus install
```

This registers Nexus as an MCP server in your Claude Code config, giving Claude access to scheduling, Discord, and Todoist tools.

Use `nexus` in place of `claude` from that point on.

## Managing secrets

API keys are stored encrypted on disk (`~/.config/nexus-ai/secrets.age`) and are only available inside the Nexus process — they are never passed to Claude.

```sh
nexus secret set <KEY> <value>
nexus secret list
nexus secret delete <KEY>
```

The encryption identity is generated automatically on first run at `~/.config/nexus-ai/identity.age`. Keep this file safe — losing it means losing access to your stored secrets.

### Available secret keys

| Key | Required | Description |
|-----|----------|-------------|
| `DISCORD_BOT_TOKEN` | For Discord | Bot token from the Discord developer portal. |
| `DISCORD_CHANNEL_ID` | For Discord | Channel where Claude posts Discord messages. |
| `DISCORD_DEBUG_CHANNEL_ID` | Optional | If set, cron job output is forwarded here after each run. |
| `TELEGRAM_BOT_TOKEN` | For Telegram | Bot token from BotFather. |
| `TELEGRAM_CHAT_ID` | For Telegram | Chat ID where Claude posts Telegram messages. |
| `TODOIST_API_KEY` | For Todoist | API token from Todoist settings. |

### Project-scoped secrets

Secrets can be scoped to a project using the `projectname::KEY` naming convention. Pass `--project <name>` to the secret commands:

```sh
nexus secret --project myapp set DB_URL postgres://localhost/myapp
nexus secret --project myapp set API_KEY secret123
nexus secret --project myapp list
nexus secret --project myapp delete DB_URL
```

This stores the secret as `myapp::DB_URL`. Global secrets (no `--project`) are still shared across all projects and used by Nexus itself (Discord, Telegram, etc.).

To inject a project secret as an environment variable when running subprocesses, configure an env mapping via the `add_project_env` MCP tool (or ask Claude to set it up):

> For project myapp, inject the secret myapp::DB_URL as DATABASE_URL when running subprocesses.

## Projects

A project maps a short name to a directory path. This lets Claude work in a specific codebase on demand without you having to specify the path every time.

### Managing projects

Projects are stored in SQLite and can be managed remotely (e.g. via Telegram) using MCP tools:

| Tool | Description |
|------|-------------|
| `add_project` | Add or update a project (`name`, `path`) |
| `delete_project` | Remove a project and its env mappings |
| `list_projects` | Show all projects with paths and env mappings |
| `add_project_env` | Map a secret key to an env var for a project |
| `delete_project_env` | Remove an env mapping |
| `switch_project` | Set the active project for this session |
| `get_current_project` | Return the currently active project |
| `run_in_project` | Run a prompt in a project's directory and return the output |

Example — ask Claude via Telegram to add a project:

> Add a project called "website" pointing to /Users/me/code/website

### Switching projects

`switch_project` sets the active project in memory and returns the project path along with a list of env vars to export. Claude uses this to `cd` into the project directory and set up the environment for the rest of the session.

To reset back to the default context, switch to an empty name:

> Switch back to no project

### Running tasks in a project

`run_in_project` spawns a `claude --print` subprocess in the project directory with all configured env vars injected. It blocks and returns the full output, so you see the result inline:

> For project myapp, run the test suite and summarise the failures.

## Cron jobs

Claude can schedule recurring tasks using the `add_cron`, `list_crons`, and `delete_cron` MCP tools. Each job runs a prompt on a cron schedule in a separate subprocess (using `claude --print`), so scheduled jobs never interrupt your active session.

Example — ask Claude to send a daily standup summary every weekday at 9am:

> Schedule a cron job that runs every weekday at 9am and asks you to summarise what I worked on yesterday and post it to Discord.

Cron expressions follow the standard 5-field format (`*/5 * * * *`) as well as descriptors like `@hourly` and `@every 30m`.

### Keeping the machine awake

Cron jobs rely on Go timers, which pause when the system sleeps. If your machine sleeps at night, scheduled jobs will be skipped rather than replayed when it wakes. On a Mac that stays plugged in (e.g. a Mac mini), disable system sleep permanently:

```sh
sudo pmset -c sleep 0
```

`-c` applies only while on AC power. The display can still sleep; only the system stays awake.

### Debug output

If `DISCORD_DEBUG_CHANNEL_ID` is configured, the full output of each cron run is forwarded to that channel after the job completes.

## Configuration

Optional, non-sensitive settings live in `~/.config/nexus-ai/config.yaml`.

```yaml
claude_args:
  - --model
  - claude-opus-4-6
```

`claude_args` are prepended to every invocation, so `nexus --verbose` would run Claude with `--model claude-opus-4-6 --verbose`.
