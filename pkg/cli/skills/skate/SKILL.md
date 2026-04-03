---
name: skate
description: Access Mattermost Boards tasks via Skate. Use to view, update, create tasks and track time on project boards.
---

# Skate — Task Management for AI Agents

You have access to project tasks on Mattermost Boards via Skate. Use these tools to stay aligned with the project plan.

## Before starting work

1. List available tasks: `skate tasks` or use `skate_tasks` tool
2. Review task details before working: `skate task <ID>` or `skate_task`
3. Update status to "In Progress": `skate update-status <ID> "In Progress"` or `skate_update_status`
4. Start time tracking: `skate timer-start <ID>` or `skate_timer_start`

## While working

- Add progress comments: `skate comment <ID> "Implemented feature X"` or `skate_comment`
- Create sub-tasks if needed: `skate create "Sub-task title"` or `skate_create_task`
- Attach files when relevant: `skate attach <ID> <file>`

## After finishing work

1. Stop timer with notes: `skate timer-stop --notes "Completed implementation"` or `skate_timer_stop`
2. Update status: `skate update-status <ID> "Done"` or `skate_update_status`
3. Add final summary comment with what was done

## Reopened / follow-up tasks

Tasks you previously completed may reappear as "Not Started" or "In Progress". This can mean:

1. **New feedback**: The user reopened the task with new comments explaining what needs to change. Read ALL comments to understand what's needed.
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
