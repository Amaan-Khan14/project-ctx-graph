package projectcontext

import (
	"testing"
	"time"
)

// ts derives fixture timestamps from merge_test.go's t1. The worked-example
// table in TASKS.md uses t2..t6 ordered ascending.
func ts(h int) time.Time { return t1.Add(time.Duration(h) * time.Hour) }

// fixtureStore builds the worked-example store from TASKS.md:
//
//	storage.format        decision    "Single JSON file, git-committed."     [.projectcontext/]  active      ev3  t5
//	merge.deterministic   decision    "Merge never infers; explicit only."   [internal/]         active      ev2  t4
//	merge.conflicts       assumption  "Conflicts will be rare at small scale." [internal/merge/] disputed    ev1  t3
//	extract.smart-caller  decision    "The calling agent performs extraction." [mcp/]            active      ev1  t6
//	extract.own-pipeline  decision    "Build our own extraction pipeline."   [mcp/]              superseded  ev1  t2
func fixtureStore() *Store {
	s := NewStore()
	add := func(key, kind, stmt, status string, scope []string, ev int, updated time.Time) {
		evs := make([]Evidence, ev)
		for i := range evs {
			evs[i] = Evidence{Session: "test", At: updated}
		}
		s.Knowledge[key] = &Knowledge{
			Key:       key,
			Kind:      kind,
			Statement: stmt,
			Scope:     scope,
			Status:    status,
			Evidence:  evs,
			Created:   t1,
			Updated:   updated,
		}
	}
	add("storage.format", "decision", "Single JSON file, git-committed.", StatusActive, []string{".projectcontext/"}, 3, ts(5))
	add("merge.deterministic", "decision", "Merge never infers; explicit only.", StatusActive, []string{"internal/"}, 2, ts(4))
	add("merge.conflicts", "assumption", "Conflicts will be rare at small scale.", StatusDisputed, []string{"internal/merge/"}, 1, ts(3))
	add("extract.smart-caller", "decision", "The calling agent performs extraction.", StatusActive, []string{"mcp/"}, 1, ts(6))
	add("extract.own-pipeline", "decision", "Build our own extraction pipeline.", StatusSuperseded, []string{"mcp/"}, 1, ts(2))
	return s
}

func requireKeys(t *testing.T, got []*Knowledge, want ...string) {
	t.Helper()
	if len(got) != len(want) {
		names := make([]string, len(got))
		for i, k := range got {
			names[i] = k.Key
		}
		t.Fatalf("want %v, got %v", want, names)
	}
	for i, k := range got {
		if k.Key != want[i] {
			t.Fatalf("position %d: want %q, got %q", i, want[i], k.Key)
		}
	}
}

// Query A: path-scoped lookup. Only merge.deterministic matches;
// merge.conflicts is excluded by the segment boundary (internal/merge/ dir
// vs internal/merge.go file), scope-less entries are excluded by the
// relevance rule.
func TestQueryPathScoped(t *testing.T) {
	got := Query(fixtureStore(), QueryOpts{Paths: []string{"internal/merge.go"}})
	requireKeys(t, got, "merge.deterministic")
}

// Query B: keyword search with superseded included. The superseded origin
// (21: two keyword hits + ev1) outranks its active replacement (11).
func TestQueryKeywordWithSuperseded(t *testing.T) {
	got := Query(fixtureStore(), QueryOpts{Text: "extraction pipeline", IncludeSuperseded: true})
	requireKeys(t, got, "extract.own-pipeline", "extract.smart-caller")
	if got[0].Status != StatusSuperseded {
		t.Fatalf("expected superseded first, got %q", got[0].Status)
	}
}

// Same text without IncludeSuperseded: the superseded entry is hidden.
func TestQueryKeywordSupersededHiddenByDefault(t *testing.T) {
	got := Query(fixtureStore(), QueryOpts{Text: "extraction pipeline"})
	requireKeys(t, got, "extract.smart-caller")
}

// Query C: kind-only list mode returns every matching entry regardless of
// score (and quietly proves disputed stays visible by default).
func TestQueryKindList(t *testing.T) {
	got := Query(fixtureStore(), QueryOpts{Kind: "assumption"})
	requireKeys(t, got, "merge.conflicts")
}

// Query D: exact-key lookup returns the entry even though it's superseded.
func TestQueryExactKeyIgnoresStatus(t *testing.T) {
	got := Query(fixtureStore(), QueryOpts{Key: "extract.own-pipeline"})
	requireKeys(t, got, "extract.own-pipeline")
	if got[0].Status != StatusSuperseded {
		t.Fatalf("expected superseded, got %q", got[0].Status)
	}
}

func TestQueryExactKeyMiss(t *testing.T) {
	if got := Query(fixtureStore(), QueryOpts{Key: "nope"}); len(got) != 0 {
		t.Fatalf("expected empty, got %d", len(got))
	}
}

// List-all with empty opts: ranked purely by evidence (no relevance terms),
// tie between the ev1 entries broken by Updated desc.
func TestQueryListAll(t *testing.T) {
	got := Query(fixtureStore(), QueryOpts{})
	requireKeys(t, got,
		"storage.format",       // ev3
		"merge.deterministic",  // ev2
		"extract.smart-caller", // ev1, updated t6
		"merge.conflicts",      // ev1, updated t3
	)
}

// A scope-less text that matches nothing yields nothing.
func TestQueryNoRelevanceExcluded(t *testing.T) {
	if got := Query(fixtureStore(), QueryOpts{Text: "zzz-no-match"}); len(got) != 0 {
		t.Fatalf("expected empty, got %d", len(got))
	}
}

// Both prefix directions count: passing the *directory* internal/ must match
// entries scoped at or below it (merge.deterministic 102, then
// merge.conflicts 101).
func TestQueryScopeBothDirections(t *testing.T) {
	got := Query(fixtureStore(), QueryOpts{Paths: []string{"internal/"}})
	requireKeys(t, got, "merge.deterministic", "merge.conflicts")
}

func TestQueryDeterministic(t *testing.T) {
	first := Query(fixtureStore(), QueryOpts{})
	for run := 0; run < 20; run++ {
		got := Query(fixtureStore(), QueryOpts{})
		if len(got) != len(first) {
			t.Fatalf("run %d: length changed", run)
		}
		for i := range got {
			if got[i].Key != first[i].Key {
				t.Fatalf("run %d: order changed at %d: %q vs %q", run, i, got[i].Key, first[i].Key)
			}
		}
	}
}

func TestScopeMatchHelper(t *testing.T) {
	cases := []struct {
		scope, path string
		want        bool
	}{
		{"internal/", "internal/merge/merge.go", true},
		{"internal/merge/", "internal/", true}, // reverse direction: caller's dir covers the scope
		{"internal/", "internal/", true},
		{"internal/merge/", "internal/merge.go", false}, // segment boundary: dir != file
		{"internal/mer/", "internal/merge/x.go", false}, // naive HasPrefix would lie
		{".projectcontext/", "internal/merge.go", false},
		{".", "internal/merge.go", false}, // root scopes match nothing
	}
	for _, c := range cases {
		if got := pathOverlap(c.scope, c.path); got != c.want {
			t.Errorf("pathOverlap(%q, %q) = %v, want %v", c.scope, c.path, got, c.want)
		}
	}
}

func TestTokenize(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"Pipeline!! EXTRACTION", []string{"pipeline", "extraction"}},
		{"extract.smart-caller", []string{"extract", "smart", "caller"}},
		{"", nil},
	}
	for _, c := range cases {
		got := tokenize(c.in)
		if len(got) != len(c.want) {
			t.Errorf("tokenize(%q) = %v, want %d tokens", c.in, got, len(c.want))
			continue
		}
		for _, w := range c.want {
			if _, ok := got[w]; !ok {
				t.Errorf("tokenize(%q) missing %q", c.in, w)
			}
		}
	}
}
