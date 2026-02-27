---
name: todo-goal-planner
use_when: User asks for an end-to-end plan toward a goal or deadline (for example plan a trip/project, build the entire plan, break this into tasks, roadmap, step-by-step, until date X), especially when research/recommendations are requested.
avoid_when: User asks only for simple create/update/delete/fetch operations on known todos, or summary-only requests.
priority: 95
tags: [todos, planning, roadmap, milestones, deadline, trip, project, research]
tools: [search, fetch_content, create_todos, fetch_todos]
---

Goal: transform a high-level goal into a practical, dated todo plan.

Rules:
1. Confirm goal scope and target date; ask one short question only if critical details are missing.
2. If user asks for web-backed suggestions, run `search` first and use `fetch_content` only for selected URLs.
3. Convert findings into actionable todos with realistic due dates leading to the target date.
4. Respect requested title prefixes or naming conventions exactly.
5. Create todos with `create_todos` using strict JSON schema.
6. Use `fetch_todos` only when needed to confirm created results or avoid duplicates.
7. Keep the response concise and practical; do not output internal tool details.

Preferred flow:
- Detect planning intent and extract goal, deadline, constraints, and requested coverage.
- Run web research when explicitly requested or required for recommendations.
- Build phased tasks (preparation, booking/setup, execution, contingency).
- Call `create_todos` in one or more valid batches.
- Confirm what was created and highlight immediate next steps.
