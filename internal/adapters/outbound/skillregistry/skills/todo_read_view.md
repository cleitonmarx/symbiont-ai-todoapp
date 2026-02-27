---
name: todo-read-view
use_when: User asks to fetch/list/show/display/find/filter/sort/paginate existing todos (for example "list my open todos", "show done tasks", "list my open todos due from March 1-7"), or asks to adjust how todos are shown (for example my screen, my list, current view, what I am seeing, shown first).
avoid_when: User asks for concise/brief summary, recap, overview, counts, paragraph-only output, or asks to create/update/reschedule/delete todos.
priority: 96
tags: [todos, read, view, filters, sorting, pagination, search, screen, list, app-view, open, done, due, due-range, date-range, from, between]
tools: [fetch_todos, set_ui_filters]
---

Goal: handle read/query and view-state intents for todos without mutating data.

Rules:
1. Use `fetch_todos` when the user asks to return todo results.
2. Use `set_ui_filters` when the user asks to apply/sync filter or sort state for their screen/list/view.
3. If user references current screen/view/list, always call `set_ui_filters`.
4. Use both tools when user asks to apply filters and then show results.
5. Keep arguments strict JSON and aligned with each tool schema.
6. If user provides relative dates (today, tomorrow, this week, next week, this month, next month), convert them to concrete YYYY-MM-DD bounds before calling `fetch_todos`.
7. When filtering by due range, always send both `due_after` and `due_before`.
8. Validate due date range coherence (`due_after` with `due_before` and `due_after <= due_before`).
8.1. Phrases like "due from March 1-7", "from March 1 to March 7", or "between March 1 and March 7" are date-range filters and must map to `due_after` and `due_before` in the same call.
9. For explicit list/show/display intents, return todo items (not just counts/summary-only text).
10. If results span multiple pages, fetch additional pages when needed to satisfy the requested listing scope.
11. Do not use this skill for create/update/delete or summary-only intents.
12. Treat natural view phrases as filter/sort intents (for example show only open, show done first, sort by due date, in my current screen/list).
13. If user requests sorting, always set `sort_by` explicitly in action arguments; never omit it.
14. Map natural sort language to schema enums exactly:
    - due date asc/oldest due first -> `dueDateAsc`
    - due date desc/latest due first/newest due first/due date DESC -> `dueDateDesc`
    - created asc/oldest created first -> `createdAtAsc`
    - created desc/newest created first/latest created first/created DESC -> `createdAtDesc`
15. Normalize status language to schema enums when filtering:
    - open -> `OPEN`
    - done/completed -> `DONE`
16. When both `set_ui_filters` and `fetch_todos` are called, keep filter/sort values consistent across both calls.
17. If the user says "related", "similar", "about", or "regarding", prefer semantic search using `search_by_similarity` (not `search_by_title`).
18. When using semantic search for related/similar intents, prefer `sort_by=similarityAsc` unless the user explicitly asked another sort.

Preferred flow:
- Detect whether intent is data retrieval, view/screen filter sync, or both.
- Build filter/sort/pagination parameters once.
- Call `set_ui_filters` if screen/list/view sync is requested (mandatory when user says current view/screen/list).
- Call `fetch_todos` when result listing/confirmation is requested.
- Return compact read-only response, but preserve itemized output when user asked to list/show tasks.
