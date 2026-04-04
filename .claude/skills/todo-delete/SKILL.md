---
name: todo-delete
description: Remove or complete a to-do item from Todoist. Use this skill whenever the user says "delete to-do", "remove to-do", "mark as done", "complete to-do", "cross off", or any similar phrase indicating they want to remove a task from their list. Also triggers on Dutch phrases like "verwijder taak", "taak verwijderen", "taak gedaan", "klaar met", or "afgevinkt". Use this skill even if the user just says something like "I finished X" or "I'm done with X" when X matches a known to-do.
---

# Todo Delete Skill

Closes (completes) a task in Todoist using `list_todoist_tasks` and `close_todoist_task` MCP tools.

## Steps

1. Call `list_todoist_tasks` to get all active tasks. If the result is empty, tell the user there are no active to-dos and stop.
2. Find the task(s) that best match the description the user gave. Use partial/fuzzy matching — the user won't always quote the item exactly. Case-insensitive matching is fine.
3. If exactly one match is found: call `close_todoist_task` with its `id` and confirm to the user, quoting the task content.
4. If multiple matches are found: show the user the matches and ask them to clarify which one they mean. Don't close anything yet.
5. If no match is found: tell the user no matching to-do was found and show them the current list so they can pick one.

## Example

Active tasks: `[{"id":"1","content":"write unit tests"},{"id":"2","content":"update README"}]`

User: "remove the to-do about unit tests"

Tool call: `close_todoist_task` with `id = "1"`

Response: "Done: write unit tests"
