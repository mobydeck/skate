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
4. Update status to "In Progress": `skate update-status <ID> "In Progress"` or `skate_update_status`
5. Start time tracking: `skate timer-start <ID>` or `skate_timer_start`

## While working

- Add progress comments: `skate comment <ID> "Implemented feature X"` or `skate_comment`
- Create sub-tasks if needed: `skate create "Sub-task title"` or `skate_create_task`
- **Attach files** to preserve context: `skate attach <ID> <file>`. Use this for test output, logs, generated configs, screenshots, or any artifact that helps the team (or future agents) understand what happened. Attachments are especially valuable for large outputs that would clutter a comment.

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

## Rules

- Always check task details and read ALL comments before starting work
- Keep task status updated as you progress
- Track time for accurate reporting
- Add comments for non-trivial changes or decisions
- When user asks to "work on a task" or "pick a task", list tasks first, then confirm which one
- Use task IDs (not titles) for all operations
- If task has sub-tasks or dependencies mentioned, review those too
- A task reappearing in the active list means the user wants action — either follow-up feedback or re-execution of a recurring task
