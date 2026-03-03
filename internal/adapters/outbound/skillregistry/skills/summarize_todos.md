---
name: todo-summary
use_when: User asks for a concise summary, recap, overview, total, count, or "how many" answer about existing todos, including when the summary is scoped by topic, status, or date window.
avoid_when: User asks to create, plan, build, generate, update, reschedule, or delete todos, explicitly asks to list/show/display individual todos, or asks to access external websites, webpages, URLs, or internet content.
priority: 100
embed_first_content_line: true
tags: [todos, summary, summarize, concise, brief, recap, overview, count, total, number, matching, matching-todos, matching-items, topic-count, filter-count, date-window-count, count-question, count-by-topic, find-and-summarize, existing-todos-summary, how-many, how-many-do-i-have, how-many-match, how-many-topic, paragraph, short, one-paragraph, due, due-range, date-window, week]
tools: [fetch_todos, execute_code]
---

Goal: summarize existing or matching todos concisely without listing individual items, using deterministic counting.

Rules:
1. Always call `fetch_todos` first.
2. Apply the user's scope on the first fetch: status, date window, and topic filter when present.
3. Do not run an unfiltered fetch when the prompt already contains a topic, status, or date constraint.
4. If the prompt is topical, prefer `search_by_similarity`; use `search_by_title` only when the user is clearly asking about title text.
5. Keep paginating until `next_page` is null. Keep the same scope on every page; only `page` changes.
6. If a scoped topical fetch returns zero results on page 1, retry once with `search_by_similarity=<topic phrase>` and `sort_by=similarityAsc`, preserving the rest of the scope.
7. Call `execute_code` after pagination before answering. Use it for exact totals and counts.
8. If the user explicitly asks for grouping or counts by category, infer exactly one short category per todo before aggregation. Use `Uncategorized` only when needed.
9. Do not answer from `fetch_todos` results alone when the user asked for a summary, total, or count. Answer only after `execute_code` returns.
10. Return one short paragraph only. Do not list individual todos or mention internal action/tool names.
11. Do not use `input[...]` or assume the interpreter injects variables automatically; inline the normalized todo list in the code.

Preferred flow:
- Build the scope.
- Fetch all pages in that scope.
- Retry once with similarity search only if the first scoped topical fetch returns zero results.
- Call `execute_code` with the accumulated normalized todos.
- Return one concise paragraph.

`execute_code` template:
```python
todos = [
    {"title": "Call Alice", "category": "Personal", "status": "OPEN"},
    {"title": "Doctor Appointment", "category": "Medical", "status": "DONE"},
    {"title": "Call my brother", "category": "Personal", "status": "DONE"},
]

by_category = {}
for todo in todos:
    category = todo.get("category") or "Uncategorized"
    item = by_category.setdefault(category, {
        "total": 0,
        "open_count": 0,
        "done_count": 0,
    })
    item["total"] += 1
    if todo.get("status") == "OPEN":
        item["open_count"] += 1
    if todo.get("status") == "DONE":
        item["done_count"] += 1

result = {
    "total": len(todos),
    "open_count": sum(1 for t in todos if t.get("status") == "OPEN"),
    "done_count": sum(1 for t in todos if t.get("status") == "DONE"),
    "by_category": by_category,
}

result
```
