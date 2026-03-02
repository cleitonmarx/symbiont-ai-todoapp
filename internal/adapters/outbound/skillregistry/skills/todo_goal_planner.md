---
name: todo-goal-planner
use_when: User asks to plan something from scratch and turn a broader goal into multiple related todos, a complete todo plan, a checklist, a roadmap, or step-by-step work toward a deadline. Also use when research, requirements, or recommendations are requested but the final deliverable should still be a plan or multiple created todos, and when a follow-up reply only provides missing planning parameters after a planning question, such as a date range, budget, location, scope, or deadline.
avoid_when: User asks to create only one concrete todo or reminder, asks to list/find/search/filter/sort/paginate/confirm existing todos, asks to find existing todos and then summarize/recap/count them, requests summary/recap/overview output as the final deliverable, asks to mark done, reopen, reschedule, delete, fetch, or otherwise update known existing todos, or is only greeting, thanking, or chatting.
priority: 94
embed_first_content_line: true
tags: [todos, plan, planning, multiple-todos, complete-todo-plan, todo-plan, roadmap, milestones, deadline, project, research, requirements, recommendations, create-plan, create-tasks, research-as-input, research-and-create-plan, research-then-plan, final-deliverable-plan, checklist, multi-step, step-by-step, parameters, date-range, budget, location, scope, follow-up]
tools: [search, fetch_content, create_todos, fetch_todos]
---

Goal: transform a broader goal or research-backed request into a practical, multi-step, dated todo plan.

Rules:
1. Confirm goal scope and target date; ask one short question only if critical details are missing.
2. If user asks for both planning and todo creation, do not stop at research-only output.
3. After gathering enough information, call `create_todos` in the same turn.
4. If user explicitly says "research first", always run `search` before creating todos.
5. Use `fetch_content` only for selected URLs that add concrete details to the plan.
6. Convert findings into actionable todos with realistic due dates; every created todo must include a valid due date.
7. Respect requested title prefixes or naming conventions exactly.
8. Do not mark newly created todos as completed/done unless the user explicitly asks.
9. Prefer robust planning tasks (scope, prerequisites, dependencies, resources, milestones, risk mitigation, review) over speculative one-off items.
10. Do not invent specific named entities or factual claims unless supported by fetched content.
11. If minor details are missing, assume sensible defaults and continue with task creation instead of blocking.
12. Create todos with `create_todos` using strict JSON schema.
13. If `create_todos` fails due to argument shape, fix and retry once.
14. Use `fetch_todos` only when needed to confirm created results or avoid duplicates.
15. Keep the response concise and practical; do not output internal tool details.
16. If the request can be satisfied by reading/filtering existing todos, do not use this skill.
17. In user-facing responses, never mention internal action/tool names (for example `search`, `fetch_content`, or `create_todos`).

Date guidance:
- If user provides an execution window, infer preparation milestones before the window and execution tasks within the window.
- Keep preparation due dates before the target window and avoid due dates after the target unless user asks for follow-up tasks.
- If the user reply only supplies missing planning parameters, continue the same planning workflow.

Preferred flow:
- Detect planning intent and extract goal, deadline, constraints, and requested coverage.
- Run web research when explicitly requested or required for recommendations.
- Build phased tasks (discovery/research, preparation, execution, verification/follow-up).
- Call `create_todos` in one or more valid batches (typically at least 5 todos for end-to-end planning, unless user asks for fewer).
- Confirm what was created and highlight immediate next steps.
