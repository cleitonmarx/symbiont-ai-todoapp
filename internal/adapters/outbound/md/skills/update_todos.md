---
name: todo-update
display_name: Update
aliases: [update, edit]
description: Edit existing todos such as title, status, or due date.
use_when: User explicitly asks to modify existing todos (update/edit/change/rename/mark complete/reopen/reschedule/postpone/change due date/change title), or clearly states that an existing todo should now have a different status/state (for example "my todo is done", "this task is completed", "reopen that todo", "my dentist todo is done", "update todo X title to Y").
avoid_when: User asks to create/add todos, fetch/list/confirm only, summarize/overview/recap/count, delete todos, or access external websites, webpages, URLs, or internet content.
priority: 90
tags: [todos, update, edit, change, rename, title-change, update-title, mutation, status, due-date, deadline, reschedule, schedule, mark, complete, completed, done, reopen, state-change, my-todo-is-done]
tools: [fetch_todos, update_todos, update_todos_due_date]
---

Goal: update existing todos safely, including both general fields and due dates.

Rules:
1. Call `fetch_todos` first.
1.1. A plain-text "updated list" response is not completion; completion requires a successful update tool call.
2. When resolving targets with `fetch_todos`, paginate all pages when needed: start at `page=1` and continue until `next_page` is null.
3. If the change is due date/deadline, prefer `update_todos_due_date`.
4. For status or title, use `update_todos`.
5. Build payloads with required schema fields.
6. Keep tool arguments as strict JSON only.
7. If update fails due to argument shape, correct and retry once.
7.1. Never claim updates were applied unless the tool result confirms success.
7.2. If update still fails, report failure clearly and ask only the minimum follow-up needed to retry.
8. Do not ask the user to wait, do not narrate that you will call tools, and do not ask for confirmation again when the user already requested the update clearly.
9. If the user confirms with a short follow-up like "yes" after you just resolved the target or proposed the exact update, treat it as approval to continue the pending update workflow.
10. When changing only the title, preserve the current due date and status unless the user explicitly asks to change them too.
11. Keywords: update, edit, change, rename, title to, mark done, complete, completed, is done, reopen, due date, deadline, reschedule, postpone.
12. If intent is read-only summary/count/overview, do not use this skill.

Preferred flow:
- Detect update intent and target todo(s).
- Always resolve IDs using `fetch_todos`; if there are multiple pages, keep fetching and accumulating matches across pages.
- If the target is unambiguous after fetch, immediately call the correct update action in the same turn.
- Route to the correct update tool (`update_todos` or `update_todos_due_date`).
- Confirm final result to the user using tool-confirmed outcomes.
