---
name: todo
description: Add a new to-do item to Todoist. Use this skill whenever the user says "make a new to-do", "add a to-do", "create a to-do", "new todo", or gives you a task to track with phrases like "remind me to", "don't forget to", or "add to my list". Also triggers on Dutch phrases like "voeg een taak toe", "nieuwe taak", "herinner me aan", or "zet op mijn lijst". Even if the user doesn't say "to-do" explicitly but clearly wants to log a task for later, use this skill.
---

# Todo Skill

Creates a new task in Todoist using the `create_todoist_task` MCP tool.

## Steps

1. Extract the task description from the user's message. Strip any leading/trailing whitespace. If the user provides a description after a colon or dash (e.g. "new todo - fix the login bug"), extract just the description part.
2. Call the `create_todoist_task` MCP tool with the description as `content`.
3. Confirm to the user that the to-do was added, quoting the description back.

## Example

User: "make a new to-do: write unit tests for the auth module"

Tool call: `create_todoist_task` with `content = "write unit tests for the auth module"`

Response: "Added to Todoist: write unit tests for the auth module"
