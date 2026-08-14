package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

const (
	markerBegin = "<!-- ctx:begin -->"
	markerEnd   = "<!-- ctx:end -->"
)

// agentsSnippet is the onboarding text init writes into a target repo's
// AGENTS.md. It is load-bearing: it teaches every future agent session the
// explore-first / record discipline without human prompting. Edit with care;
// it must stay tool-agnostic (MCP tools, CLI, or raw JSON fallback).
var agentsSnippet = markerBegin + `
## Project knowledge (ctx)

This repo accumulates project understanding — decisions, constraints, bugs,
assumptions, rationale — in ` + "`.ctx/knowledge.json`" + `. It exists so you do not
re-derive what is already known.

**Read first.** Before planning or editing unfamiliar areas, query it:
- MCP tools: ` + "`ctx_explore`" + ` (query for topics, paths for files you will touch,
  key for exact lookup, include_superseded for history).
- CLI: ` + "`ctx explore --query <topic>`" + ` or ` + "`ctx explore --path <paths>`" + `.
- Fallback: read ` + "`.ctx/knowledge.json`" + ` directly — it is plain JSON.
Statuses matter: ` + "`superseded`" + ` = historical position, kept for history — do not
follow it; ` + "`disputed`" + ` = contested — weigh in or proceed carefully. Scope ` + "`.`" + `
means project-wide.

**Record what you learn.** When you discover something non-obvious — a
decision and its why, a constraint, a bug's root cause, an assumption —
record it so the next session does not rediscover it:
- MCP tools: ` + "`ctx_record`" + `; CLI: ` + "`ctx record --key K --kind K --statement S --scope P`" + `.
- Key: a stable dot.case slug naming the TOPIC, e.g. ` + "`storage.format`" + `.
  One decision per key.
- Kind: decision | constraint | bug | assumption | rationale | fact.
- Statement: the current position, 1-2 sentences.
- Scope: paths it applies to (` + "`.`" + ` for project-wide).
- Do not duplicate: explore first; same topic -> record again with the SAME
  key (re-observation strengthens it). A decision replacing an earlier one ->
  new key + ` + "`supersedes`" + `.
- ` + "`ctx_dispute`" + ` flags knowledge as contested; it is not a correction mechanism —
  correct by recording.
` + markerEnd + `
`

// ensureAgentsSnippet makes sure repoDir/AGENTS.md carries the ctx snippet.
// Kept for the init path; equivalent to ensureMarkdownSnippet(dir, "AGENTS.md").
func ensureAgentsSnippet(repoDir string) (string, error) {
	return ensureMarkdownSnippet(repoDir, "AGENTS.md")
}

// ensureMarkdownSnippet makes sure dir/<name> carries the ctx snippet.
// It never clobbers existing content. Returns the outcome: "present",
// "created", or "appended".
func ensureMarkdownSnippet(dir, name string) (string, error) {
	path := filepath.Join(dir, name)

	data, err := os.ReadFile(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		if err := os.WriteFile(path, []byte(agentsSnippet), 0o644); err != nil {
			return "", fmt.Errorf("creating AGENTS.md: %w", err)
		}
		return "created", nil
	case err != nil:
		return "", fmt.Errorf("reading AGENTS.md: %w", err)
	}

	if bytes.Contains(data, []byte(markerBegin)) {
		return "present", nil
	}

	sep := []byte("\n\n")
	if len(bytes.TrimSpace(data)) == 0 {
		sep = nil // empty file: snippet becomes the whole content
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return "", fmt.Errorf("opening AGENTS.md: %w", err)
	}
	defer f.Close()
	if _, err := f.Write(append(sep, []byte(agentsSnippet)...)); err != nil {
		return "", fmt.Errorf("appending to AGENTS.md: %w", err)
	}
	return "appended", nil
}
