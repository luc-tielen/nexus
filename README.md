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

## Cron jobs

Claude can schedule recurring tasks using the `add_cron`, `list_crons`, and `delete_cron` MCP tools. Each job runs a prompt on a cron schedule in a separate subprocess (using `claude --print`), so scheduled jobs never interrupt your active session.

Example — ask Claude to send a daily standup summary every weekday at 9am:

> Schedule a cron job that runs every weekday at 9am and asks you to summarise what I worked on yesterday and post it to Discord.

Cron expressions follow the standard 5-field format (`*/5 * * * *`) as well as descriptors like `@hourly` and `@every 30m`.

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
