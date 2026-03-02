---
name: web-research
use_when: User asks for external information, references, recent updates, requirements, rankings, recommendations, or content from websites, webpages, the internet, or explicit URLs (for example "research the web", "research the internet", "look up current rules", "open this website", "fetch this webpage", "tell me the page title", "search the web for sources", "find the best options online", "research the top options online", "research the top 3 options online", "look up current requirements online", "find the best places online").
avoid_when: Request is fully about internal todo CRUD with no need for external sources, or asks to use research as an input step inside a larger plan, checklist, roadmap, or task breakdown instead of as the final answer. If research is only one step and the requested final deliverable is a plan or created tasks, prefer the planning skill (for example "research and create a plan" or "research and create tasks").
priority: 40
embed_first_content_line: true
tags: [web, website, webpage, page, page-title, url, external-url, search, fetch, content, research, online, internet, references, requirements, look-up, sources, external, recommendations, ranking, best, top, top-n, browse, current-info, latest, top-3]
tools: [search, fetch_content]
---

Goal: access external websites or search sources safely and answer with fetched web information.

Rules:
1. Start with `search` to find relevant sources before fetching pages.
2. Use focused queries and keep `max_results` small unless user asks for broad research.
3. Use `fetch_content` when the user explicitly asks to open/fetch/read a specific webpage or URL, or when a concrete URL is needed for deeper details.
4. Prefer one URL fetch per turn unless user asks to compare multiple sources.
5. Keep tool arguments strict JSON and aligned with schema.

Preferred flow:
- Detect external-info intent.
- Call `search` with a focused query.
- Select the best source URL.
- Call `fetch_content` when deeper evidence is needed.
- Respond with concise findings and source links.
