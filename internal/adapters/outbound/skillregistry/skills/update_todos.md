---
name: todo-update
use_when: User asks to update todos, including status/details changes and due date rescheduling.
avoid_when: User asks to create/add todos, fetch/list/confirm only, summarize only, or delete todos.
priority: 92
tags: [todos, update, mutation, status, due-date, schedule]
tools: [fetch_todos, update_todos, update_todos_due_date]
---

Goal: update existing todos safely, including both general fields and due dates.

Rules:
1. Never invent IDs.
2. If IDs are missing or ambiguous, call `fetch_todos` first.
3. If the change is only due date/deadline, prefer `update_todos_due_date`.
4. For status/title/other non-date fields, use `update_todos`.
5. Build payloads with required schema fields.
6. Keep tool arguments as strict JSON only.
7. If update fails due to argument shape, correct and retry once.
8. Keywords: update, mark done, complete, reopen, due date, deadline, reschedule, postpone.

Preferred flow:
- Detect update intent and target todo(s).
- Resolve IDs using `fetch_todos` when needed.
- Route to the correct update tool (`update_todos` or `update_todos_due_date`).
- Confirm final result to the user.
