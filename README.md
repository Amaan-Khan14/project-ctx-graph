# ctx — accumulated project context for coding sessions

`ctx` gives a project a persistent, structured memory. Decisions, constraints,
bugs, assumptions, rationale, and facts are recorded as typed **knowledge
entries** in a single git-committed JSON file, so any future session — human or
agent — can ask *what does this project already know?* before acting.

The philosophy: a **dumb store with careful bookkeeping, driven by smart
callers**. The tool never infers, auto-resolves, or deletes anything; callers
(humans or coding agents) explicitly record, update, supersede, and dispute.

## Core ideas

- **Nothing is ever deleted.** Wrong or outdated entries are *superseded* by
  their replacement, or *disputed* to flag them for human review. History is
  preserved and re-surfaceable.
- **Fully deterministic merge.** Same store + same operation = same result.
  No inference, no auto-resolution, no fuzzy matching.
- **Git is the history of record.** The store is one indented JSON file with
  sorted keys and atomic writes, so diffs stay clean and reviewable.
- **Scoped knowledge.** Every entry names the paths it applies to (`scope`),
  so querying "I'm about to touch `internal/merge/`" surfaces exactly the
  constraints that govern that area. Scope `.` means project-wide.
- **Zero dependencies.** Go standard library only.

## Data model

Each **Knowledge** entry has:

| field | meaning |
|---|---|
| `key` | stable caller-chosen slug naming the *topic* (e.g. `merge.deterministic`); one position per key |
| `kind` | `decision` \| `constraint` \| `bug` \| `assumption` \| `rationale` \| `fact` |
| `statement` | the current position on the topic; updated in place as understanding evolves |
| `scope` | paths this knowledge applies to; `.` = project-global |
| `status` | `active` \| `superseded` \| `disputed` |
| `evidence` | observations backing the entry (session, note, timestamp); confidence is *derived* from evidence, never stored |

Supersession is recorded as a typed `supersedes` **edge**, so a query with
`--include-superseded` can show an old position side-by-side with the one that
replaced it.

## Installation

### For CLI users

Requires Go 1.22+.

```sh
# Install directly from GitHub (recommended)
go install github.com/yourusername/ctx/cmd/ctx@latest

# Or build from source
git clone https://github.com/yourusername/ctx.git
cd ctx
go build -o ctx ./cmd/ctx
```

### For MCP server (AI agents)

No manual installation needed! Configure your MCP client:

```json
{
  "mcpServers": {
    "ctx": {
      "command": "go",
      "args": ["run", "github.com/yourusername/ctx/cmd/ctx@latest", "serve"]
    }
  }
}
```

On first use, Go will automatically download and cache the server.

## Usage

All commands operate on `.ctx/knowledge.json` in the current directory.

### `ctx init`

Initialize a knowledge store.

```sh
ctx init            # creates .ctx/knowledge.json; refuses if it exists
ctx init --force    # overwrite an existing store
```

### `ctx record`

Record knowledge, or add evidence to an existing key.

```sh
ctx record --key storage.format --kind decision \
  --statement "Single JSON file, git-committed." \
  --scope .

# Confirming/refining an existing topic: record the same key again.
# The statement updates in place and evidence accumulates.

# Replacing an old decision: record the new key and supersede the old one.
ctx record --key extract.smart-caller --kind decision \
  --statement "The calling agent performs extraction." \
  --scope mcp/ --supersedes extract.own-pipeline
```

Flags: `--key --kind --statement --scope a,b [--supersedes k1,k2] [--session] [--note]`
(`--scope` and `--supersedes` are comma-separated; `--session` defaults to `cli`).

### `ctx dispute`

Flag an entry as contested when you believe it's wrong but can't yet replace
it. Disputed entries **stay visible** — that's the point: a human should weigh
in. To *correct* knowledge, use `record` instead (same key, or new key +
`--supersedes`).

```sh
ctx dispute --key merge.conflicts --note "keys keep colliding"
```

### `ctx explore`

Retrieve knowledge — the read path, and the actual point of the tool.

```sh
ctx explore                                        # list everything (ranked)
ctx explore --query "extraction pipeline"          # keyword search
ctx explore --path internal/merge/merge.go         # knowledge scoped to a path
ctx explore --kind decision                        # filter by kind
ctx explore --key storage.format                   # exact lookup (even superseded)
ctx explore --include-superseded                   # reveal historical positions
ctx explore --json                                 # machine-readable output
```

Ranking is deterministic: scope matches outweigh keyword matches, evidence
count breaks ties, then recency, then key. Same store + same query = same
output, every time.

Example human output:

```
merge.deterministic  [decision]  (evidence: 2)
  Merge never infers; explicit only.
  scope: internal/  updated: 2026-08-06

merge.conflicts  [assumption, DISPUTED]  (evidence: 1)
  Conflicts will be rare at small scale.
  scope: internal/merge/  updated: 2026-08-05
```

## Project layout

```
├── cmd/ctx/          # the ctx CLI (flag-based subcommands, one binary)
│   └── main.go  init.go  record.go  dispute.go  explore.go
├── types.go          # Knowledge / Evidence / Edge / Store / QueryOpts
├── store.go          # Load/Save: atomic writes, diff-stable JSON
├── merge.go          # Record/Dispute: the deterministic merge rules
├── query.go          # retrieval: filter → score → sort
└── *_test.go
```

The root package `projectcontext` holds all core semantics; the CLI is a thin
presentation layer over it. Retrieval contract:

```go
func Query(s *Store, opts QueryOpts) []*Knowledge
```

## Roadmap

- **M1/M2 — done.** Core package (store, deterministic merge, query) and the
  `ctx` CLI (`init`, `record`, `dispute`, `explore`), dogfooded on this repo's
  own `.ctx/` store.
- **M3 — next.** `ctx serve`: an MCP stdio server exposing `ctx_record`,
  `ctx_explore`, and `ctx_dispute` as tools so coding agents can read and
  write project knowledge natively.
- **M4.** Onboarding: `init` bootstraps an AGENTS.md snippet into the target
  repo.

## Development

```sh
gofmt -l .        # must be clean
go vet ./...      # must be clean
go test ./...     # must be green
```
