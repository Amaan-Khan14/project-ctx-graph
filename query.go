package codedocket

import (
	"sort"
	"strings"
	"unicode"
)

// Ranking weights encode strict priority: scope > keyword > evidence.
// Any scope match beats keyword-only matches; evidence only nudges.
const (
	weightScope    = 100
	weightKeyword  = 10
	weightEvidence = 1
)

// Query retrieves knowledge: filter -> score -> sort. It is deterministic —
// the total-order sort makes Go's random map iteration irrelevant.
func Query(s *Store, opts QueryOpts) []*Knowledge {
	// Exact-key lookup is precise retrieval: status filters do not apply,
	// and superseded entries are returned when asked for by key.
	if opts.Key != "" {
		if k, ok := s.Knowledge[opts.Key]; ok {
			return []*Knowledge{k}
		}
		// Always an empty slice, never nil: JSON renderers must see [].
		return []*Knowledge{}
	}

	// When the caller gave Text or Paths, an entry needs a nonzero relevance
	// part (scope or keyword match) to be returned at all. Evidence only
	// ranks; it never qualifies.
	scoped := opts.Text != "" || len(opts.Paths) > 0

	type scored struct {
		k     *Knowledge
		score int
	}
	var cands []scored

	for _, k := range s.Knowledge {
		// Filter: disputed stays visible on purpose; superseded is hidden
		// unless the caller asked for it.
		if k.Status == StatusSuperseded && !opts.IncludeSuperseded {
			continue
		}
		if opts.Kind != "" && k.Kind != opts.Kind {
			continue
		}

		relevance := weightScope*scopeMatches(k.Scope, opts.Paths) +
			weightKeyword*keywordOverlap(opts.Text, k)
		if scoped && relevance == 0 {
			continue
		}
		cands = append(cands, scored{k, relevance + weightEvidence*len(k.Evidence)})
	}

	sort.Slice(cands, func(i, j int) bool {
		a, b := cands[i], cands[j]
		if a.score != b.score {
			return a.score > b.score
		}
		if !a.k.Updated.Equal(b.k.Updated) {
			return a.k.Updated.After(b.k.Updated)
		}
		return a.k.Key < b.k.Key // total order => deterministic output
	})

	out := make([]*Knowledge, len(cands))
	for i, c := range cands {
		out[i] = c.k
	}
	return out
}

// scopeMatches counts (scope, path) pairs where one side is a segment-wise
// path-prefix of the other. Both directions count: an entry scoped to a
// directory matches a file in it, and an entry scoped to a file matches a
// directory the caller is about to work in. A scope of "." (or "/") is
// project-global and matches every path.
func scopeMatches(scope, paths []string) int {
	n := 0
	for _, sc := range scope {
		if sc == "." || sc == "./" || sc == "/" {
			n += len(paths) // global scope matches every path
			continue
		}
		for _, p := range paths {
			if pathOverlap(sc, p) {
				n++
			}
		}
	}
	return n
}

// pathOverlap reports whether a and b share a segment-wise prefix
// relationship. Empty segment lists (".", "/", "") match nothing: scopes
// must name real directories.
func pathOverlap(a, b string) bool {
	as, bs := pathSegments(a), pathSegments(b)
	n := min(len(as), len(bs))
	if n == 0 {
		return false
	}
	for i := 0; i < n; i++ {
		if as[i] != bs[i] {
			return false
		}
	}
	return true
}

func pathSegments(s string) []string {
	parts := strings.Split(s, "/")
	segs := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" && p != "." {
			segs = append(segs, p)
		}
	}
	return segs
}

// keywordOverlap is the set-intersection size of the tokenized query text
// against the entry's pooled key+statement tokens. No stemming:
// "extract" != "extraction" (documented V1 limitation).
func keywordOverlap(text string, k *Knowledge) int {
	if text == "" {
		return 0
	}
	want := tokenize(text)
	have := tokenize(k.Key + " " + k.Statement)
	n := 0
	for tok := range want {
		if _, ok := have[tok]; ok {
			n++
		}
	}
	return n
}

func tokenize(s string) map[string]struct{} {
	m := make(map[string]struct{})
	for _, f := range strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}) {
		m[f] = struct{}{}
	}
	return m
}
