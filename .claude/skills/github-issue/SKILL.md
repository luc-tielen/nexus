---
name: github-issue
description: Create a new GitHub issue in the luc-tielen/nexus repository. Use this skill whenever the user says "create an issue", "open an issue", "new GitHub issue", "file a bug", "report a bug", "add an issue", or any similar phrase. Also triggers on Dutch: "maak een issue aan", "nieuw issue", "bug melden". Use this skill even when the user describes a problem or feature request and it's clear they want it tracked on GitHub.
---

# GitHub Issue Skill

Creates a new issue in the `luc-tielen/nexus` GitHub repository using the `gh` CLI.

## Steps

1. Identify the issue title from what the user said — keep it concise.
2. Write the issue body:
   - Start with the user's exact description.
   - If you have relevant context from the current conversation (e.g. related code, recent changes, error messages, file paths, or background on the feature), expand the body with that information to make the issue more useful. Structure it clearly — e.g. **Description**, **Context**, **Steps to reproduce**, **Expected behavior** — only including sections that have real content.
   - If you have no additional context beyond what the user said, keep the body minimal. Don't pad it with generic filler.
3. Determine the appropriate label(s) if obvious from context (e.g. "bug", "enhancement", "documentation"). Skip labels if unclear — don't guess.
4. Run:
   ```bash
   gh issue create --repo luc-tielen/nexus --title "<title>" --body "<body>"
   ```
   Add `--label <label>` for each label if applicable.
5. Report the issue URL back to the user.

## Example

User: "create an issue: the CI workflow fails when there are no test files"

```bash
gh issue create \
  --repo luc-tielen/nexus \
  --title "CI workflow fails when there are no test files" \
  --body "" \
  --label "bug"
```

## Notes

- The `gh` CLI must be authenticated (`gh auth login`). If it's not, tell the user to run `gh auth login` first.
- Keep titles concise and descriptive.
- If the user provides a multi-line description, use it as the body verbatim.
