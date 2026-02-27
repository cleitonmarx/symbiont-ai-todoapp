---
name: todo-summary
use_when: User asks to summarize/recap/overview/count todos in compact form (for example "give me a concise summary of my medical appointments", "make a concise summary of open todos due from March 1-7", "summary in one short paragraph", "brief summary", "high-level recap").
avoid_when: User asks to create, update, reschedule, or delete todos, or explicitly asks to list/show/display individual todos.
priority: 100
tags: [todos, summary, summarize, concise, brief, recap, overview, count, paragraph, short, one-paragraph, medical, appointments, due, due-range, date-window, week]
tools: [fetch_todos]
---

Goal: provide a concise summary without listing individual todos.

Rules:
1. Always call `fetch_todos` first.
2. If user provides a date window (explicit or relative), apply `due_after` and `due_before` on the first fetch.
3. If the date window is explicit and valid, do not ask follow-up questions.
4. If user mentions a topic/domain (for example medical appointments, tax, travel), include a query filter on the first fetch (`search_by_similarity` preferred; `search_by_title` acceptable when explicitly title-oriented).
5. Do not run unfiltered fetches when the prompt contains topical constraints.
6. Pagination is mandatory for summaries: if `next_page` is not null, call `fetch_todos` again with that page.
7. Keep paginating until `next_page` is null. Do not produce the final summary before the loop is complete.
8. Keep the same scope across pagination (status, due range, query, sort); only `page` changes.
9. If the prompt is topical and the first scoped fetch returns zero results, retry once with `search_by_similarity=<topic phrase>` and `sort_by=similarityAsc`, preserving status and due window.
10. Aggregate all pages before summarizing.
11. Return one short paragraph only (no category breakdown, no itemized list).
12. Treat phrases like "medical appointments", "doctor visits", "health tasks", and "due from <month day-day>" as strong summary filters, not as follow-up questions.

Preferred flow:
- Build scope first (status + due window + topic query when present).
- Fetch page 1.
- If `next_page` exists, keep fetching `page=next_page` with the same scope until `next_page` is null.
- If zero results on page 1, retry once with similarity search in the same scope.
- Return one concise paragraph summary.
