# Nexus

A personal AI assistant, powered by Claude Code.

## Installation

```sh
nexus install
```

This registers Nexus as an MCP server in your Claude Code config, giving Claude access to scheduling, Discord, Todoist, and project tools.

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

Secrets can be scoped to a project using the `projectname::KEY` naming convention. Pass `--project <name>` to the secret commands (the project must already exist):

```sh
nexus secret --project myapp set DATABASE_URL postgres://localhost/myapp
nexus secret --project myapp set API_KEY secret123
nexus secret --project myapp list
nexus secret --project myapp delete DATABASE_URL
```

The key is stored as `myapp::DATABASE_URL`. When a subprocess runs in a project context, all of that project's secrets are automatically injected as environment variables — the prefix is stripped, so `myapp::DATABASE_URL` becomes `DATABASE_URL` in the subprocess.

Global secrets (no `--project`) are used by Nexus itself (Discord, Telegram, etc.) and are never injected into project subprocesses.

## Projects

A project maps a short name to a directory path. This lets Claude work in a specific codebase on demand without you having to specify the path every time.

### Managing projects (CLI)

Projects are created and deleted from the CLI — they are setup-time config, not runtime state:

```sh
nexus project add <name> <path>    # ~ is expanded; errors if name already exists
nexus project delete <name>        # interactive confirmation; also deletes project secrets
nexus project list
```

Example:

```sh
nexus project add myapp ~/code/myapp
nexus project add website ~/code/website
```

### Using projects (MCP)

Once created, projects can be used from within a Claude session via MCP tools:

| Tool | Description |
|------|-------------|
| `list_projects` | Show all configured projects with their paths |
| `switch_project` | Set the active project for this session |
| `get_current_project` | Return the currently active project |
| `run_in_project` | Run a prompt in a project's directory and return the output |

**Switching projects**

`switch_project` sets the active project in memory and returns the project path along with a list of env vars to export. Claude uses this to `cd` into the project directory and set up the environment for the rest of the session. Pass an empty name to reset.

**Running tasks in a project**

`run_in_project` spawns a `claude --print` subprocess in the project directory with all project-scoped secrets injected as environment variables. It blocks and returns the full output inline:

> For project myapp, run the test suite and summarise the failures.

**Deleting projects remotely**

To delete a project from a Claude session (e.g. via Telegram), use `delete_project`. Without `confirm: true` it returns a dry-run summary; with `confirm: true` it deletes the project and all its secrets.

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
# Enable a communication channel. Supported values: telegram
# The required Claude plugin args are added to the main interactive session only —
# background subprocesses (cron jobs, run_in_project) do NOT get them, which
# prevents multiple competing bot listeners from being spawned.
channel: telegram

# Extra arguments prepended to every claude invocation.
claude_args:
  - --model
  - claude-opus-4-6
```

`claude_args` are prepended to every invocation, so `nexus --verbose` would run Claude with `--model claude-opus-4-6 --verbose`.

### Telegram channel

Set `channel: telegram` to connect the main Claude session to your Telegram bot. This automatically adds `--channels plugin:telegram@claude-plugins-official` to the interactive process. Make sure the `TELEGRAM_BOT_TOKEN` and `TELEGRAM_CHAT_ID` secrets are configured.
