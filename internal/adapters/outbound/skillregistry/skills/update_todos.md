---
name: todo-update
use_when: User explicitly asks to modify existing todos (update/edit/change/mark complete/reopen/reschedule/postpone/change due date), or clearly states that an existing todo should now have a different status/state (for example "my todo is done", "this task is completed", "reopen that todo", "my dentist todo is done").
avoid_when: User asks to create/add todos, fetch/list/confirm only, summarize/overview/recap/count, delete todos, or access external websites, webpages, URLs, or internet content.
priority: 90
tags: [todos, update, mutation, status, due-date, schedule, mark, complete, completed, done, reopen, state-change, my-todo-is-done]
tools: [fetch_todos, update_todos, update_todos_due_date]
---

Goal: update existing todos safely, including both general fields and due dates.

Rules:
1. Never invent IDs.
2. If IDs are missing or ambiguous, call `fetch_todos` first.
3. When resolving targets with `fetch_todos`, paginate all pages when needed: start at `page=1` and continue until `next_page` is null.
4. If the change is only due date/deadline, prefer `update_todos_due_date`.
5. For status/title/other non-date fields, use `update_todos`.
6. Build payloads with required schema fields.
7. Keep tool arguments as strict JSON only.
8. If update fails due to argument shape, correct and retry once.
9. Keywords: update, mark done, complete, completed, is done, reopen, due date, deadline, reschedule, postpone.
10. If intent is read-only summary/count/overview, do not use this skill.

Preferred flow:
- Detect update intent and target todo(s).
- Resolve IDs using `fetch_todos` when needed; if there are multiple pages, keep fetching and accumulating matches across pages.
- Route to the correct update tool (`update_todos` or `update_todos_due_date`).
- Confirm final result to the user.
