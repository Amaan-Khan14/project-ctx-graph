package codedocket

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	MaxKeyLen       = 120
	MaxStatementLen = 500
)

// EdgeSupersedes is the only edge type interpreted by V1.
const EdgeSupersedes = "supersedes"

var keyRe = regexp.MustCompile(`^[a-z0-9]+([.-][a-z0-9]+)*$`)

// RecordInput is one caller observation of project knowledge.
type RecordInput struct {
	Key        string
	Kind       string
	Statement  string
	Scope      []string
	Session    string // defaults to "unknown" when empty
	Note       string // evidence note for this observation
	Supersedes []string
}

// Record applies the deterministic merge rules:
//
//  1. unknown key -> insert knowledge with first evidence
//  2. known key   -> append evidence; statement/kind update in place;
//     scope unions additively; Updated stamps
//  3. Supersedes  -> mark targets superseded, add supersedes edges
//
// All validation happens before any mutation; an error leaves the store
// untouched.
func Record(s *Store, in RecordInput, now time.Time) (*Knowledge, bool, error) {
	if len(in.Key) > MaxKeyLen || !keyRe.MatchString(in.Key) {
		return nil, false, fmt.Errorf("invalid key %q: must match %s and be <= %d chars", in.Key, keyRe, MaxKeyLen)
	}
	if !validKind(in.Kind) {
		return nil, false, fmt.Errorf("invalid kind %q: must be one of %s", in.Kind, strings.Join(Kinds, ", "))
	}
	stmt := strings.TrimSpace(in.Statement)
	if stmt == "" {
		return nil, false, fmt.Errorf("statement must not be empty")
	}
	if len(stmt) > MaxStatementLen {
		return nil, false, fmt.Errorf("statement too long: %d chars, max %d (split into multiple keys)", len(stmt), MaxStatementLen)
	}

	targets := make([]string, 0, len(in.Supersedes))
	seen := map[string]bool{}
	for _, t := range in.Supersedes {
		if t == in.Key {
			return nil, false, fmt.Errorf("a key cannot supersede itself (%q)", t)
		}
		if seen[t] {
			continue
		}
		seen[t] = true
		if _, ok := s.Knowledge[t]; !ok {
			return nil, false, fmt.Errorf("supersedes target %q does not exist", t)
		}
		targets = append(targets, t)
	}

	ev := Evidence{Session: orDefault(in.Session, "unknown"), Note: in.Note, At: now}

	k, ok := s.Knowledge[in.Key]
	created := !ok
	if ok {
		if k.Status == StatusSuperseded {
			return nil, false, fmt.Errorf("key %q is superseded; record under a new key", in.Key)
		}
		k.Kind = in.Kind
		k.Statement = stmt
		k.Scope = unionScope(k.Scope, in.Scope)
		k.Evidence = append(k.Evidence, ev)
		k.Updated = now
	} else {
		k = &Knowledge{
			Key:       in.Key,
			Kind:      in.Kind,
			Statement: stmt,
			Scope:     unionScope(nil, in.Scope),
			Status:    StatusActive,
			Evidence:  []Evidence{ev},
			Created:   now,
			Updated:   now,
		}
		s.Knowledge[in.Key] = k
	}

	for _, t := range targets {
		tgt := s.Knowledge[t]
		tgt.Status = StatusSuperseded
		tgt.Updated = now
		if !edgeExists(s.Edges, EdgeSupersedes, in.Key, t) {
			s.Edges = append(s.Edges, Edge{Type: EdgeSupersedes, From: in.Key, To: t, At: now})
		}
	}
	return k, created, nil
}

// Dispute marks an entry as disputed. Explicit, caller-driven, never
// inferred. Superseded entries cannot be disputed; dispute the successor.
func Dispute(s *Store, key, session, note string, now time.Time) (*Knowledge, error) {
	k, ok := s.Knowledge[key]
	if !ok {
		return nil, fmt.Errorf("key %q does not exist", key)
	}
	if k.Status == StatusSuperseded {
		return nil, fmt.Errorf("key %q is superseded; dispute the successor instead", key)
	}
	k.Status = StatusDisputed
	k.Evidence = append(k.Evidence, Evidence{Session: orDefault(session, "unknown"), Note: note, At: now})
	k.Updated = now
	return k, nil
}

func validKind(k string) bool {
	for _, v := range Kinds {
		if k == v {
			return true
		}
	}
	return false
}

// unionScope merges two scope lists, dedupes, and sorts for deterministic
// output. Empty entries are dropped; no other path normalization happens.
func unionScope(existing, add []string) []string {
	set := map[string]bool{}
	out := make([]string, 0, len(existing)+len(add))
	for _, p := range existing {
		if !set[p] {
			set[p] = true
			out = append(out, p)
		}
	}
	for _, p := range add {
		p = strings.TrimSpace(p)
		if p == "" || set[p] {
			continue
		}
		set[p] = true
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

func edgeExists(edges []Edge, typ, from, to string) bool {
	for _, e := range edges {
		if e.Type == typ && e.From == from && e.To == to {
			return true
		}
	}
	return false
}

func orDefault(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}
