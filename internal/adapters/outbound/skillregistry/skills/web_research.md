---
name: web-research
use_when: User asks for external information, references, recent updates, requirements, rankings, recommendations, or content from websites or URLs (for example "research Japan visa requirements", "research the web", "research the internet", "look up current rules", "search the web for sources", "find the best options online", "research the top options online").
avoid_when: Request is fully about internal todo CRUD with no need for external sources, or asks to use research as an input step inside a larger plan, checklist, roadmap, or task breakdown instead of as the final answer. If research is only one step and the requested final deliverable is a plan or created tasks, prefer the planning skill (for example "research and create a plan" or "research and create tasks").
priority: 20
tags: [web, search, fetch, content, research, online, internet, references, requirements, look-up, sources, external, recommendations, ranking, best, top, top-n]
tools: [search, fetch_content]
---

Goal: gather external information safely and provide source-backed answers.

Rules:
1. Start with `search` to find relevant sources before fetching pages.
2. Use focused queries and keep `max_results` small unless user asks for broad research.
3. Use `fetch_content` only when a concrete URL is needed for deeper details.
4. Prefer one URL fetch per turn unless user asks to compare multiple sources.
5. Keep tool arguments strict JSON and aligned with schema.

Preferred flow:
- Detect external-info intent.
- Call `search` with a focused query.
- Select the best source URL.
- Call `fetch_content` when deeper evidence is needed.
- Respond with concise findings and source links.
