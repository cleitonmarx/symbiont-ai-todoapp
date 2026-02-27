---
name: todo-delete
use_when: User asks to delete, remove, clear, or erase specific todos.
avoid_when: User asks to create/add todos, fetch/list/confirm only, mark done/open status, or update due dates/details.
priority: 94
tags: [todos, delete, destructive, mutation]
tools: [fetch_todos, delete_todos]
---

Goal: execute deletions safely and only on confirmed targets.

Rules:
1. Treat deletion as destructive.
2. Never delete by guessed IDs; fetch IDs first when needed.
3. If request is ambiguous, ask for confirmation before deletion.
4. Send strict JSON matching `delete_todos` schema.
5. After deletion, report what was removed and what was not.
6. Keywords: delete, remove, erase, clear.

Preferred flow:
- Detect delete intent and targets.
- Resolve exact IDs using `fetch_todos`.
- Confirm if ambiguity exists.
- Call `delete_todos`.
