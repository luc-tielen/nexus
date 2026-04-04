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
nexus secret set TODOIST_API_KEY <key>
nexus secret set DISCORD_BOT_TOKEN <token>
nexus secret set DISCORD_CHANNEL_ID <channel-id>
nexus secret list
nexus secret delete TODOIST_API_KEY
```

The encryption identity is generated automatically on first run at `~/.config/nexus-ai/identity.age`. Keep this file safe — losing it means losing access to your stored secrets.

## Configuration

Optional, non-sensitive settings live in `~/.config/nexus-ai/config.yaml`.

```yaml
claude_args:
  - --model
  - claude-opus-4-6
```

`claude_args` are prepended to every invocation, so `nexus --verbose` would run Claude with `--model claude-opus-4-6 --verbose`.
