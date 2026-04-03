---
name: skate
description: Access Mattermost Boards tasks via Skate. Use to view, update, create tasks and track time on project boards.
---

# Skate — Task Management for AI Agents

You have access to project tasks on Mattermost Boards via Skate. Use these tools to stay aligned with the project plan.

## Before starting work

1. List available tasks: `skate tasks` or use `skate_tasks` tool
2. Review task details before working: `skate task <ID>` or `skate_task` (shows last 5 comments; use `--full` for all)
   - To see all comments only: `skate comments <ID>`
3. **Check for attached files**: use `skate task-files <ID>` to list attachments. If the task has images, text, markdown, config files, or other readable files — download and review them before starting. They often contain essential context, screenshots of bugs, or reference material.
   - Download: `skate download <FILE_ID> -o filename.ext`
4. **Search for related tasks** if you need context: `skate find "keyword"` or `skate_find` MCP tool. Searches titles first, then content blocks and comments. Useful for finding prior work, related bugs, or duplicate tasks before starting.
5. Update status to "In Progress": `skate update-status <ID> "In Progress"` or `skate_update_status`
6. Start time tracking: `skate timer-start <ID>` or `skate_timer_start`

## While working

- Add progress comments: `skate comment <ID> "Implemented feature X"` or `skate_comment`
- Create sub-tasks if needed: `skate create "Sub-task title"` or `skate_create_task`
- **Attach files** to preserve context: `skate attach <ID> <file>`. Use this for test output, logs, generated configs, screenshots, or any artifact that helps the team (or future agents) understand what happened. Attachments are especially valuable for large outputs that would clutter a comment.

## Content blocks vs comments

Tasks have two types of text: **content blocks** (the Description section) and **comments**.

**Comments** are for work summaries, progress updates, and communication. They're timestamped, attributed, and shown in chronological order. Use `skate comment` or `skate_comment`.

**Content blocks** are the card's persistent body — the Description section. Use them for long-term reference material: discoveries, architectural decisions, important findings, code patterns, or anything future agents/humans should know when revisiting this task.

**Available block types:**
```bash
skate add-content <ID> "Some reference notes"              # text (default)
skate add-content <ID> "Architecture" -t h2                # heading (h1, h2, h3)
skate add-content <ID> -t divider                          # horizontal divider
skate add-content <ID> "Review security audit" -t checkbox # checkbox item
skate add-content <ID> ./diagram.png -t image              # inline image (uploads file)
```
MCP: `skate_add_content` with `block_type` parameter (text, h1, h2, h3, divider, checkbox, image). For image, pass the **absolute** file path as `text` to avoid working directory issues.

**When to add content blocks:**
- You discovered something non-obvious about the code while working on a task
- The user pointed out context, constraints, or decisions worth preserving
- You want to document a pattern, workaround, or edge case for future reference
- There's a diagram or visual that helps understand the task — add a text block with description, then an image block

**Format for text content blocks:**
Start each text block with your signature and timestamp so it's clear who added it and when:
```
— claude-code (claude-opus-4-6) | 2026-04-03

Discovered that the plugin adapter hardcodes all board-scoped WS messages
as UPDATE_BOARD events. The actual action type is inside the payload.
```

## Mentions in comments

By default, mention the last relevant person when adding comments — this notifies them in Mattermost. Use `@username` at the start of your comment text.

**Who to mention:**
- If no comments exist yet: mention the **task creator** (shown in task detail output)
- If comments exist: mention the **last commenter** (the most recent comment author)
- Only mention **one person** — the last relevant one, not everyone

**Example:**
```
skate comment <ID> "@arthur Fixed the UTF-8 truncation bug. — claude-code (claude-opus-4-6)"
```

**Disabling mentions:**
Mentions can be disabled per project via config (`mentions: false` in `.skate.yaml`). Check with `skate config` to see the effective setting. If mentions are disabled, skip the `@username` prefix entirely.

## After finishing work

1. Stop timer with notes: `skate timer-stop --notes "Completed implementation"` or `skate_timer_stop`
2. Update status: `skate update-status <ID> "Done"` or `skate_update_status`
3. Add final summary comment with what was done

## When to block a task

If a task involves major changes, ambiguous requirements, or missing information — do not guess. Instead:

1. Add a comment explaining what you need: a decision, clarification, data, or user feedback
2. Attach any relevant files that show the current state or the problem
3. Update status to "Blocked": `skate update-status <ID> "Blocked"`
4. Stop the timer: `skate timer-stop --notes "Blocked: waiting for ..."`

The user will review, respond, and reopen the task when ready. This is better than making the wrong assumption and having to redo the work.

## Reopened / follow-up tasks

Tasks you previously completed may reappear as "Not Started" or "In Progress". This can mean:

1. **New feedback**: The user reopened the task with new comments explaining what needs to change. Use `skate comments <ID>` or `skate task <ID> --full` to read ALL comments.
2. **Repetitive task**: Some tasks are recurring by nature (e.g., "Update docs", "Run tests", "Update README"). The user wants you to execute the task again with fresh context — even if there are no new comments. Check the current state of the relevant files and act accordingly.

In both cases: update status, start timer, do the work, then complete again.

## Signature

Always append a short signature line at the end of every comment and timer note so it's clear which agent/model produced it. Format:

```
— <agent> (<model>)
```

Examples:
- `— claude-code (claude-opus-4-6)`
- `— codex (gpt-5-codex)`
- `— cursor (gpt-4o)`

This is for informational purposes only — helps the team understand which AI contributed what.

## Plans (IMPORTANT)

If a task requires planning and you produce a plan, you **MUST** attach it to the task as a markdown file:

- File name: `plan-<short-description>.md` (e.g., `plan-refactor-config.md`)
- Attach with: `skate attach <ID> plan-<name>.md`
- If the plan is modified during work, attach each revision
- **The final version of the plan MUST be attached before completing the task.** This is non-negotiable. The user relies on plan attachments to understand what was decided and why. Missing a final plan attachment is a deal breaker.

## Rules

- Always check task details and read ALL comments before starting work
- Keep task status updated as you progress
- Track time for accurate reporting
- Add comments for non-trivial changes or decisions
- When user asks to "work on a task" or "pick a task", list tasks first, then confirm which one
- Use task IDs (not titles) for all operations
- If task has sub-tasks or dependencies mentioned, review those too
- A task reappearing in the active list means the user wants action — either follow-up feedback or re-execution of a recurring task
