---
name: todo
description: Add a new to-do item to the project's todos.md file. Use this skill whenever the user says "make a new to-do", "add a to-do", "create a to-do", "new todo", or gives you a task to track with phrases like "remind me to", "don't forget to", or "add to my list". Also triggers on Dutch phrases like "voeg een taak toe", "nieuwe taak", "herinner me aan", or "zet op mijn lijst". Even if the user doesn't say "to-do" explicitly but clearly wants to log a task for later, use this skill.
---

# Todo Skill

Adds a new to-do item to `todos.md` in the project root, creating the file if it doesn't exist yet.

## Steps

1. Get today's date from the system (use `date +%Y-%m-%d` via Bash, or use the current date from context if available).
2. Check whether `todos.md` exists in the project root.
   - If it doesn't exist, create it with this header:
     ```
     # To-Do
     ```
3. Append a new checkbox line at the end of `todos.md`:
   ```
   - [ ] <description> _(YYYY-MM-DD)_
   ```
4. Confirm to the user that the to-do was added, quoting the description back.

## Example

User: "make a new to-do: write unit tests for the auth module"

Appended line:
```
- [ ] write unit tests for the auth module _(2026-03-28)_
```

## Notes

- Never overwrite existing items — always append.
- Strip any leading/trailing whitespace from the description before writing.
- If the user provides a description after a colon or dash (e.g. "new todo - fix the login bug"), extract just the description part.
