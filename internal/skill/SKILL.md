---
name: skate
description: Access Mattermost Boards tasks via Skate. Use to view, update, create tasks and track time on project boards.
---

# Skate — Task Management for AI Agents

## Board ID

Most MCP tools that touch a board accept a `board_id` arg. The server's configured default (if any) is shown at the top of `skate_help`. If you see `board_id required`, call `skate_boards` and pass the right ID. CLI users in a project with `.skate.yaml` don't need to think about this.

## Before starting work

1. **Resuming a session?** `skate state` (or `skate_state`) shows your running timer and In Progress tasks — call this first when picking up where you left off.
2. **Quick pick:** `skate next` (or `skate_next`, optionally `--mine`) returns the top-priority Not Started task with full details. Use when the user says "work on next task" or "pick the top task" — saves a list+sort+pick round-trip.
3. Otherwise: `skate tasks` to list, then `skate task <ID>` for full detail (last 5 comments; `--full` for all, or `skate comments <ID>` for comments only).
4. **Check attached files**: `skate task-files <ID>`. Download anything readable — bug screenshots, configs, logs, prior plans often hold essential context.
   - CLI: `skate download <FILE_ID> -o filename.ext`
   - MCP: `skate_download` returns small text files (≤32 KiB, valid UTF-8) inline. Larger or binary files are auto-saved to `~/.cache/skate/downloads/<file_id>` and the path is returned — read it with your own file tools. Pass `output_path` to choose the location yourself. Clean up later with `skate cache clean`.
5. **Search for related tasks** when you need context: `skate find "keyword"` / `skate_find` searches titles, content blocks, and comments.
6. **Check valid statuses** before changing one: `skate statuses` / `skate_statuses`. Names vary per board ("Completed 🙌" vs "Done") — do NOT guess.
7. Set status + start timer in one call: `skate update-status <ID> "In Progress" --timer` or `skate_update_status` with `start_timer: true`.

## Working on a task

- **Comments** are timestamped, attributed, chronological — for progress updates and communication. Use `skate comment` / `skate_comment`.
- **Content blocks** are the persistent Description body — for discoveries, decisions, code patterns, anything future agents/humans should know on revisit. Use `skate add-content` / `skate_add_content`.
- **Attachments** preserve artifacts that would clutter a comment (test output, logs, generated configs, screenshots). Use `skate attach <ID> <file>` / `skate_attach` with an absolute `file_path`.
- **Sub-tasks**: `skate create` / `skate_create_task`.

### Content block types

```bash
skate add-content <ID> "Some reference notes"              # text (default)
skate add-content <ID> "Architecture" -t h2                # heading (h1, h2, h3)
skate add-content <ID> -t divider                          # horizontal divider
skate add-content <ID> "Review security audit" -t checkbox # checkbox item
skate add-content <ID> ./diagram.png -t image              # inline image (uploads file)
```

MCP: `skate_add_content` with `block_type`. For images, pass an **absolute** file path as `text`.

Start text blocks with a signature + timestamp:
```
— claude-code (claude-opus-4-7) | 2026-04-29
Plugin adapter hardcodes board-scoped WS messages as UPDATE_BOARD. Real action type lives in the payload.
```

### Fixing mistakes

Wrong comment, outdated block, accidental attachment? Find the block ID via `skate task <ID> --json` (look at `blocks[]`) or `skate task-files <ID>`. Then:
- Rewrite in place: `skate edit-block <TASK_ID> <BLOCK_ID> "new text"` / `skate_edit_block`
- Remove entirely: `skate delete-block <TASK_ID> <BLOCK_ID>` / `skate_delete_block` (also cleans the card's content order)

### Renaming, reassigning, re-prioritizing

`skate update <ID> --title ... --priority ... --assignee ... --icon ...` or `skate_update_task`. `update-status` is a shortcut for the status-only case.

## Mentions and signatures

Mention the last relevant person in every comment — that's what triggers their Mattermost/email notification. Use `@username` at the start.

- No comments yet → mention the **task creator** (shown in task detail).
- Comments exist → mention the **last commenter**.
- Only one person — the last relevant one, not everyone.
- It is fine — and often desired — to mention the same Mattermost user this token is authenticated as. Agents commonly share an account with their human operator, so the @-mention is what delivers the notification. Use `skate me` / `skate_me` to confirm; don't suppress mentions just because the names match.
- Don't have the username? `skate users <substring>` / `skate_users` resolves names to handles.

**Always sign every comment and timer note** so it's clear which agent/model produced it: `— <agent> (<model>)`, e.g. `— claude-code (claude-opus-4-7)`.

Example:
```
skate comment <ID> "@arthur Fixed the UTF-8 truncation bug. — claude-code (claude-opus-4-7)"
```

Mentions can be disabled per project (`mentions: false` in `.skate.yaml` — check with `skate config`). When disabled, skip the `@username` prefix; still include the signature.

## Finishing work

1. Stop timer with notes: `skate timer-stop --notes "..."` / `skate_timer_stop`.
2. Set final status: `skate update-status <ID> "Done"` (call `skate_statuses` first if unsure).
3. Add a final summary comment.

## Blocking a task

Don't guess on ambiguous requirements or missing info. Instead:
1. Comment explaining what you need (decision, clarification, data, user feedback).
2. Attach anything that shows the current state or the problem.
3. `skate update-status <ID> "Blocked"`.
4. `skate timer-stop --notes "Blocked: waiting for ..."`.

The user reviews and reopens when ready. This beats wrong assumptions and rework.

## Reopened tasks

A previously-completed task reappearing as Not Started or In Progress means one of:
1. **New feedback** — read ALL comments (`skate comments <ID>` or `skate task <ID> --full`).
2. **Recurring task** (e.g. "Update docs", "Run tests") — execute again with fresh context, no new comments needed.

Either way: status, timer, work, complete.

## Plans (IMPORTANT)

If you produce a plan, you **MUST** attach the final version to the task before marking it complete. Missing plan attachments break the user's ability to review what was decided.

- Filename: `plan-<short-description>.md` (e.g. `plan-refactor-config.md`)
- Attach: `skate attach <ID> plan-<name>.md`
- If the plan changes during work, attach the new revision too — the latest attachment is canonical.

## Rules

- Read ALL comments before starting; keep status current as you progress.
- Track time for accurate reporting.
- Comment on non-trivial changes or decisions.
- When the user says "work on a task" / "pick a task", list and confirm before acting.
- Use task IDs (not titles) in tool args.
- Review sub-tasks and dependencies the task mentions.
