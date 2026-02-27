---
name: todo-read-view
use_when: User asks to fetch/list/find/search/filter/sort/paginate/check/confirm existing todos, or asks to adjust how todos are shown (for example my screen, my list, current view, what I am seeing, shown first).
avoid_when: User asks to create, update, reschedule, summarize, or delete todos.
priority: 93
tags: [todos, read, view, filters, sorting, pagination, search, screen, list, app-view]
tools: [fetch_todos, set_ui_filters]
---

Goal: handle read/query and view-state intents for todos without mutating data.

Rules:
1. Use `fetch_todos` when the user asks to return todo results.
2. Use `set_ui_filters` when the user asks to apply/sync filter or sort state for their screen/list/view.
3. Use both tools when user asks to apply filters and then show results.
4. Keep arguments strict JSON and aligned with each tool schema.
5. Validate due date range coherence (`due_after` with `due_before`).
6. Do not use this skill for create/update/delete or summary-only intents.
7. Treat natural view phrases as filter/sort intents (for example show only open, show done first, sort by due date, in my current screen/list).

Preferred flow:
- Detect whether intent is data retrieval, view/screen filter sync, or both.
- Build filter/sort/pagination parameters once.
- Call `set_ui_filters` if screen/list/view sync is requested.
- Call `fetch_todos` when result listing/confirmation is requested.
- Return compact read-only response.
