---
name: todo-summary
use_when: User asks to summarize/recap/overview/count todos in compact form, or asks for the count/total/number of todos matching a topic, filter, or date window (for example "give me a concise summary", "make a concise summary of open todos due from March 1-7", "summary in one short paragraph", "brief summary", "high-level recap", "how many matching todos do I have", "how many todos match this topic", "how many of those do I have", "what is the total", "how many items do I have about this topic", "how many appointments do I have").
avoid_when: User asks to create, update, reschedule, or delete todos, or explicitly asks to list/show/display individual todos.
priority: 100
embed_first_content_line: true
tags: [todos, summary, summarize, concise, brief, recap, overview, count, total, number, matching, matching-todos, matching-items, topic-count, filter-count, date-window-count, count-question, count-by-topic, how-many, how-many-do-i-have, how-many-match, how-many-topic, paragraph, short, one-paragraph, due, due-range, date-window, week]
tools: [fetch_todos, execute_code]
---

Goal: provide a concise summary without listing individual todos, using deterministic counting when available.

Rules:
1. Always call `fetch_todos` first.
2. If user provides a date window (explicit or relative), apply `due_after` and `due_before` on the first fetch.
3. If the date window is explicit and valid, do not ask follow-up questions.
4. If user mentions a topic, theme, or matching constraint, include a query filter on the first fetch (`search_by_similarity` preferred; `search_by_title` acceptable when explicitly title-oriented).
5. Do not run unfiltered fetches when the prompt contains topical constraints.
6. Pagination is mandatory for summaries: if `next_page` is not null, call `fetch_todos` again with that page.
7. Keep paginating until `next_page` is null. Do not produce the final summary before the loop is complete.
8. Keep the same scope across pagination (status, due range, query, sort); only `page` changes.
9. If the prompt is topical and the first scoped fetch returns zero results, retry once with `search_by_similarity=<topic phrase>` and `sort_by=similarityAsc`, preserving status and due window.
10. Aggregate all pages before summarizing.
11. Use `execute_code` after pagination to compute exact counters from the accumulated todos.
12. Pass only normalized fields into `execute_code` (for example `category`, `status`) plus the final accumulated list and requested scope.
13. Use `execute_code` for deterministic totals, counts by status or category if needed, and validation that the final total matches the number of accumulated todos.
14. Do not ask the model to do math that `execute_code` can do deterministically.
15. Return one short paragraph only (no category breakdown, no itemized list).
16. Treat topical phrases and explicit date windows as strong summary filters, not as follow-up questions.
17. Do not use `input[...]` or assume the interpreter injects a payload variable automatically.
18. Unless the `execute_code` tool schema explicitly provides a variables/input field, inline the normalized todo list directly in the Python code.
19. If the user explicitly asks for grouping or counts by category, infer exactly one category per todo before aggregation.
20. Use short, practical inferred categories based on the todo title; if a clear category cannot be inferred, use `Uncategorized`.
21. When inferring categories, do not place the same todo in more than one category; category totals must sum exactly to the total number of todos in scope.

Preferred flow:
- Build scope first (status + due window + topic query when present).
- Fetch page 1.
- If `next_page` exists, keep fetching `page=next_page` with the same scope until `next_page` is null.
- If zero results on page 1, retry once with similarity search in the same scope.
- If the user asked for grouping by category, infer one category per todo before running deterministic aggregation.
- If `execute_code` is available, send the accumulated normalized todos to it and get back exact counters for the final response.
- Return one concise paragraph summary using the deterministic counter result.

`execute_code` template:
```python
todos = [
    {"category": "Personal", "status": "OPEN"},
    {"category": "Medical", "status": "DONE"},
    {"category": "Personal", "status": "DONE"},
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
