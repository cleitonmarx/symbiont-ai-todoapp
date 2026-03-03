---
name: todo-create
use_when: User asks to create/add/new todos or reminders, especially a single concrete todo with a clear title, short instruction, or direct due date (for example create todo, add task, remind me to, create one todo, create a todo named X due tomorrow).
avoid_when: User asks to fetch/list/confirm existing todos, mark done/open, change only due dates, delete todos, asks for an end-to-end plan, roadmap, checklist, or multi-step breakdown toward a broader goal, or asks to inspect/open/fetch/read an external website, webpage, URL, or internet source.
priority: 92
tags: [todos, create, add, new, reminder, single-item, direct-create, concrete-title, due-date, mutation]
tools: [create_todos]
---

Goal: create todos with complete and valid payloads.

Rules:
1. Use `create_todos` for all creation intents.
2. Include all required fields for each item in the `todos` array.
3. Keep tool arguments strict JSON matching the schema.
4. If due dates are ambiguous, ask one short follow-up question before creating.
5. If tool call fails due to argument shape, fix and retry once.
6. Keywords: create, add, new, reminder, create one todo, create a todo named, due tomorrow.
7. In user-facing responses, never mention internal action/tool names (for example `create_todos`).

Preferred flow:
- Detect creation intent and extract todo items.
- Normalize titles and dates to schema format.
- Call `create_todos` with validated input.
- Confirm how many todos were created.
