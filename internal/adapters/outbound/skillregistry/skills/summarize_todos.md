---
name: todo-summary
use_when: User asks for a concise summary/overview/recap todos, including prompts like "don't list todos", "summary only", or "how many todos".
avoid_when: User asks to create, update, reschedule, delete, or explicitly list individual todos.
priority: 96
tags: [todos, summary, concise, recap, overview, count]
tools: [fetch_todos]
---

Goal: provide a concise summary of todos without listing individual tasks.

Rules:
1. Always fetch todos first with `fetch_todos`.
2. Paginate through all results: start with `page=1`, keep fetching with `next_page` until it is null.
3. Accumulate todos from every fetched page before producing the final summary.
4. Count the total number of fetched todos across all pages.
5. Ensure the total count matches the accumulated fetched results.
6. If counts do not reconcile, recompute and fix before answering.
7. Do not output category breakdowns or individual task lists.
8. If the user asks for similarity search, include similarity parameters in `fetch_todos`; otherwise use normal filters.
9. Keep the explanation very compact.
10. This skill overrides default task-list formatting rules for this turn.
11. Use one concise sentence to describe the overall focus/trend.

Counting algorithm:
1. Initialize `total = 0`.
2. Iterate all fetched todos once.
3. Increment `total` by 1 for each todo.
4. If `total` differs from accumulated fetched results metadata, recalculate before responding.

Output format:
- `Total Summary (N tasks)`
- One short sentence with the main focus/trend.

Preferred flow:
- Fetch first page for the requested scope.
- While `next_page` is present, fetch the next page and accumulate items.
- Validate reconciliation for the final total.
- Return only the final compact summary.
