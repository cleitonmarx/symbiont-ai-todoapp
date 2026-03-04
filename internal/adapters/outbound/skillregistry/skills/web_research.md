---
name: web-research
use_when: User asks for external information, references, recent updates, requirements, rankings, recommendations, or content from websites, webpages, the internet, or explicit URLs (for example "research the web", "research the internet", "look up current rules", "open this website", "fetch this webpage", "tell me the page title", "search the web for sources", "find the best options online", "research the top options online", "research the top 3 options online", "look up current requirements online", "find the best places online", "research the top options in a place", "find the best hotels in a city online"), or asks to inspect a specific external page and extract information from it.
avoid_when: Request is fully about internal todo CRUD with no need for external sources, or asks to use research as an input step inside a larger plan, checklist, roadmap, or task breakdown instead of as the final answer. If research is only one step and the requested final deliverable is a plan or created tasks, prefer the planning skill (for example "research and create a plan" or "research and create tasks").
priority: 60
embed_first_content_line: true
tags: [web, website, webpage, page, page-title, url, external-url, search, fetch, content, research, online, internet, references, requirements, look-up, sources, external, recommendations, ranking, best, top, top-n, browse, current-info, latest, top-3, place, location, city, options-in-location]
tools: [search, fetch_content]
---

Goal: access external websites or search sources safely and answer only from fetched web information, not from memory.

Rules:
1. If the user provides a concrete external URL and asks about that page, skip `search` and call `fetch_content` first.
2. For page-title requests, fetch the page first and extract the title from fetched content.
3. For explicit webpage or URL inspection requests, do not answer until `fetch_content` succeeds or fails in this turn.
4. Do not answer questions about a specific webpage or URL from prior knowledge or memory.
5. If the page was not fetched in this turn, do not present the page title or page content as confirmed.
6. Do not say you cannot access webpages or cannot fetch URLs when this skill is active; use the available web tools instead.
7. Words like "exact", "read first", "open this page", "fetch this page", or "do not guess" make page fetch mandatory.
8. Use `search` only when no concrete URL was provided and you need to find relevant sources first.
9. Use focused queries and keep `max_results` small unless user asks for broad research.
10. Prefer one URL fetch per turn unless user asks to compare multiple sources.
11. Keep tool arguments strict JSON and aligned with schema.
12. In user-facing responses, never mention internal action/tool names (for example `search` or `fetch_content`).

Preferred flow:
- Detect external-info intent.
- If the user already gave a concrete URL and asked about that page, call `fetch_content` first.
- If the user asks for the page title, fetch the page first and answer only with the confirmed title from fetched content.
- Otherwise call `search` with a focused query.
- Select the best source URL.
- Call `fetch_content` when page-level evidence is needed.
- Respond with concise findings and source links.
