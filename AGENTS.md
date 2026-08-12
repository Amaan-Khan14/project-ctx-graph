<!-- ctx:begin -->
## Project knowledge (ctx)

This repo accumulates project understanding — decisions, constraints, bugs,
assumptions, rationale — in `.ctx/knowledge.json`. It exists so you do not
re-derive what is already known.

**Read first.** Before planning or editing unfamiliar areas, query it:
- MCP tools: `ctx_explore` (query for topics, paths for files you will touch,
  key for exact lookup, include_superseded for history).
- CLI: `ctx explore --query <topic>` or `ctx explore --path <paths>`.
- Fallback: read `.ctx/knowledge.json` directly — it is plain JSON.
Statuses matter: `superseded` = historical position, kept for history — do not
follow it; `disputed` = contested — weigh in or proceed carefully. Scope `.`
means project-wide.

**Record what you learn.** When you discover something non-obvious — a
decision and its why, a constraint, a bug's root cause, an assumption —
record it so the next session does not rediscover it:
- MCP tools: `ctx_record`; CLI: `ctx record --key K --kind K --statement S --scope P`.
- Key: a stable dot.case slug naming the TOPIC, e.g. `storage.format`.
  One decision per key.
- Kind: decision | constraint | bug | assumption | rationale | fact.
- Statement: the current position, 1-2 sentences.
- Scope: paths it applies to (`.` for project-wide).
- Do not duplicate: explore first; same topic -> record again with the SAME
  key (re-observation strengthens it). A decision replacing an earlier one ->
  new key + `supersedes`.
- `ctx_dispute` flags knowledge as contested; it is not a correction mechanism —
  correct by recording.
<!-- ctx:end -->
