---
name: todo-delete
description: Remove or complete a to-do item from the project's todos.md file. Use this skill whenever the user says "delete to-do", "remove to-do", "mark as done", "complete to-do", "cross off", or any similar phrase indicating they want to remove a task from their list. Also triggers on Dutch phrases like "verwijder taak", "taak verwijderen", "taak gedaan", "klaar met", or "afgevinkt". Use this skill even if the user just says something like "I finished X" or "I'm done with X" when X matches a known to-do.
---

# Todo Delete Skill

Removes a to-do item from `todos.md` in the project root.

## Steps

1. Read `todos.md`. If it doesn't exist, tell the user there are no to-dos and stop.
2. Find the line(s) that best match the description the user gave. Use partial/fuzzy matching — the user won't always quote the item exactly. Case-insensitive matching is fine.
3. If exactly one match is found: remove that line from the file and confirm to the user, quoting the item that was removed.
4. If multiple matches are found: show the user the matches and ask them to clarify which one they mean. Don't delete anything yet.
5. If no match is found: tell the user no matching to-do was found and show them the current list so they can pick one.

## Example

todos.md before:
```
# To-Do
- [ ] write unit tests for the auth module _(2026-03-28)_
- [ ] update README _(2026-03-28)_
```

User: "remove the to-do about unit tests"

todos.md after:
```
# To-Do
- [ ] update README _(2026-03-28)_
```

Response: "Removed: write unit tests for the auth module"

## Notes

- Only remove unchecked (`- [ ]`) items by default. If the user specifically asks to remove a checked item, do so.
- Don't leave stray blank lines after removal — keep the file tidy.
- If todos.md ends up with only the header and no items after deletion, that's fine — leave the header in place.
