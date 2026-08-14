// Package codedocket accumulates project understanding across coding
// sessions: a dumb store with careful bookkeeping, driven by smart callers.
package codedocket

import "time"

// Status values for a Knowledge entry. Nothing is ever deleted; entries are
// downgraded instead.
const (
	StatusActive     = "active"
	StatusSuperseded = "superseded"
	StatusDisputed   = "disputed"
)

// Kinds lists the valid Kind values for a Knowledge entry.
var Kinds = []string{"decision", "constraint", "bug", "assumption", "rationale", "fact"}

// Knowledge is a single unit of project understanding, addressed by a
// stable caller-chosen key. The key identifies the topic; the statement
// is the current position on that topic and may be updated in place.
type Knowledge struct {
	Key       string     `json:"key"`
	Kind      string     `json:"kind"`
	Statement string     `json:"statement"`
	Scope     []string   `json:"scope"`
	Status    string     `json:"status"`
	Evidence  []Evidence `json:"evidence"`
	Created   time.Time  `json:"created"`
	Updated   time.Time  `json:"updated"`
}

// Evidence is one observation backing a Knowledge entry. Confidence is
// derived from evidence; it is never stored directly.
type Evidence struct {
	Session string    `json:"session"`
	Note    string    `json:"note,omitempty"`
	At      time.Time `json:"at"`
}

// Edge is a typed relationship between two Knowledge keys. V1 interprets
// only "supersedes"; other types are stored verbatim for forward compat.
type Edge struct {
	Type string    `json:"type"`
	From string    `json:"from"`
	To   string    `json:"to"`
	At   time.Time `json:"at"`
}

// CurrentVersion is the on-disk schema version of knowledge.json.
const CurrentVersion = 1

// Store is the materialized state of knowledge.json.
type Store struct {
	Version   int                   `json:"version"`
	Knowledge map[string]*Knowledge `json:"knowledge"`
	Edges     []Edge                `json:"edges"`
}

// QueryOpts describes a retrieval request against a Store.
type QueryOpts struct {
	Text              string   // keyword match against key + statement
	Paths             []string // files/dirs the caller is about to touch
	Kind              string   // "" = any
	Key               string   // set => exact lookup, returns at most 1 result
	IncludeSuperseded bool     // default: active + disputed only
}

// NewStore returns an empty store at the current schema version.
func NewStore() *Store {
	return &Store{Version: CurrentVersion, Knowledge: map[string]*Knowledge{}, Edges: []Edge{}}
}
