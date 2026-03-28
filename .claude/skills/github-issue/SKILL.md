---
name: github-issue
description: Create a new GitHub issue in the luc-tielen/nexus repository. Use this skill whenever the user says "create an issue", "open an issue", "new GitHub issue", "file a bug", "report a bug", "add an issue", or any similar phrase. Also triggers on Dutch: "maak een issue aan", "nieuw issue", "bug melden". Use this skill even when the user describes a problem or feature request and it's clear they want it tracked on GitHub.
---

# GitHub Issue Skill

Creates a new issue in the `luc-tielen/nexus` GitHub repository using the `gh` CLI.

## Steps

1. Identify the issue title and body from what the user said.
   - If they gave enough detail, use it directly.
   - If only a brief description was given, use it as the title and leave the body minimal.
2. Determine the appropriate label(s) if obvious from context (e.g. "bug", "enhancement", "documentation"). Skip labels if unclear — don't guess.
3. Run:
   ```bash
   gh issue create --repo luc-tielen/nexus --title "<title>" --body "<body>"
   ```
   Add `--label <label>` for each label if applicable.
4. Report the issue URL back to the user.

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
