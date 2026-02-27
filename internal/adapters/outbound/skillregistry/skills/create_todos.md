---
name: todo-create
use_when: User asks to create/add/new todos or reminders (for example create todo, add task, remind me to).
avoid_when: User asks to fetch/list/confirm existing todos, mark done/open, change only due dates, or delete todos.
priority: 92
tags: [todos, create, mutation, planning]
tools: [create_todos]
---

Goal: create todos with complete and valid payloads.

Rules:
1. Use `create_todos` for all creation intents.
2. Include all required fields for each item in the `todos` array.
3. Keep tool arguments strict JSON matching the schema.
4. If due dates are ambiguous, ask one short follow-up question before creating.
5. If tool call fails due to argument shape, fix and retry once.
6. Keywords: create, add, new, reminder, plan.

Preferred flow:
- Detect creation intent and extract todo items.
- Normalize titles and dates to schema format.
- Call `create_todos` with validated input.
- Confirm how many todos were created.
