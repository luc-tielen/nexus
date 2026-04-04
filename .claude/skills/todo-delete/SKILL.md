---
name: todo-delete
description: Remove or complete a to-do item. Use this skill whenever the user says "delete to-do", "remove to-do", "mark as done", "complete to-do", "cross off", or any similar phrase indicating they want to remove a task from their list. Also triggers on Dutch phrases like "verwijder taak", "taak verwijderen", "taak gedaan", "klaar met", or "afgevinkt". Use this skill even if the user just says something like "I finished X" or "I'm done with X" when X matches a known to-do.
---

# Todo Delete Skill

Completes a task using the `list_tasks` and `complete_task` MCP tools.

Supports two lookup modes:
- **By name** (default): list tasks, find the closest match, confirm with user before acting.
- **By ID**: skip listing, confirm with user before acting.

## Steps

### Deleting by name (default)

1. Call `list_tasks` to retrieve all active tasks. If the result is empty, tell the user there are no active to-dos and stop.
2. Find the task(s) that best match the name the user provided. Use partial/fuzzy, case-insensitive matching — the user won't always quote the item exactly.
3. If exactly one match is found: show the task name and details to the user and ask for confirmation before proceeding. Only call `complete_task` once the user confirms.
4. If multiple matches are found: show the matches and ask the user to clarify which one they mean. Do not delete anything yet.
5. If no match is found: tell the user and show the current task list so they can pick one.

### Deleting by ID

1. If the user explicitly provides a task ID, skip the listing step.
2. Show the task ID (and any other known details) to the user and ask for confirmation before proceeding.
3. Only call `complete_task` once the user confirms.

## Confirmation rule

**Always ask for confirmation before any deletion.** The confirmation prompt must include the task name/content so the user knows exactly what will be removed.

## Example — by name

Active tasks: `[{"id":"1","content":"write unit tests"},{"id":"2","content":"update README"}]`

User: "remove the to-do about unit tests"
