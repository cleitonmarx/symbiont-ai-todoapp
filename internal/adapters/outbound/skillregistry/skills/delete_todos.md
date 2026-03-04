---
name: todo-delete
use_when: User asks to delete, remove, clear, or erase todos, whether they name specific todos or refer to a subset of todos selected by a natural-language description, phrase, topic, status, criteria, search-like filter, or group/set description (for example delete old matching todos).
avoid_when: User asks to create/add todos, plan/build/generate a checklist or roadmap, fetch/list/show/display/find/search/filter/confirm only, asks for related/similar/about/regarding todos without destructive intent, mark done/open status, update due dates/details, asks to inspect/open/fetch/read an external website, webpage, URL, or internet source, or is only greeting, thanking, or chatting.
priority: 94
embed_first_content_line: true
tags: [todos, delete, remove, clear, erase, destructive, mutation, subset, natural-language-selection, matching-todos, filtered-todos, selected-set, selected-by-description, matching-set, group-delete, matching-description, matching-topic, matching-phrase, criteria, search-filter, status-delete]
tools: [fetch_todos, delete_todos]
---

Goal: delete one or more existing todos, including subsets selected by natural-language description or filter, such as old or completed matching items, safely and only on confirmed targets.

Rules:
1. Treat deletion as destructive.
2. Never delete by guessed IDs; fetch IDs first when needed.
3. When resolving targets with `fetch_todos`, paginate all pages when needed: start at `page=1` and continue until `next_page` is null.
4. If user provided explicit target titles and fetched matches are unambiguous, proceed directly to `delete_todos` in the same turn.
5. If request is ambiguous, ask for confirmation before deletion.
6. Send strict JSON matching `delete_todos` schema.
7. If first delete attempt fails due to missing/mismatched IDs, fetch again to resolve IDs and retry once in the same turn.
8. Do not ask the user to wait, do not narrate that you will call tools, and do not ask for confirmation again when the target and delete intent are already clear.
9. If the user confirms with a short follow-up like "yes" after you just resolved the target or proposed the exact deletion, treat it as approval to continue the pending delete workflow.
10. After deletion, report what was removed and what was not.
11. Keywords: delete, remove, erase, clear.

Preferred flow:
- Detect delete intent and targets.
- Resolve exact IDs using `fetch_todos`; if there are multiple pages, keep fetching and accumulating matches across pages.
- Confirm if ambiguity exists.
- Call `delete_todos` immediately after IDs are resolved.
- Return final deletion result (not an "I will do it next" message).
