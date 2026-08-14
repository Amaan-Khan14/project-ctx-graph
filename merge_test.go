package codedocket

import (
	"strings"
	"testing"
	"time"
)

var (
	t1 = time.Date(2026, 8, 6, 10, 0, 0, 0, time.UTC)
	t2 = t1.Add(time.Hour)
)

func rec(key, kind, stmt string) RecordInput {
	return RecordInput{Key: key, Kind: kind, Statement: stmt, Scope: []string{"internal/"}, Session: "test"}
}

func TestRecordCreates(t *testing.T) {
	s := NewStore()
	k, created, err := Record(s, rec("merge.rules", "decision", "Merge is deterministic."), t1)
	if err != nil {
		t.Fatalf("Record: %v", err)
	}
	if !created {
		t.Fatal("expected created=true")
	}
	if k.Status != StatusActive || len(k.Evidence) != 1 || k.Evidence[0].Session != "test" {
		t.Fatalf("bad initial state: %+v", k)
	}
	if !k.Created.Equal(t1) || !k.Updated.Equal(t1) {
		t.Fatalf("timestamps: %+v", k)
	}
	if s.Knowledge["merge.rules"] != k {
		t.Fatal("store not updated")
	}
}

func TestRecordValidation(t *testing.T) {
	cases := map[string]RecordInput{
		"uppercase key": rec("Merge.Rules", "decision", "x"),
		"space in key":  rec("merge rules", "decision", "x"),
		"key too long":  rec(strings.Repeat("a", MaxKeyLen+1), "decision", "x"),
		"invalid kind":  rec("k", "opinion", "x"),
		"empty stmt":    rec("k", "decision", "   "),
		"stmt too long": rec("k", "decision", strings.Repeat("x", MaxStatementLen+1)),
	}
	for name, in := range cases {
		s := NewStore()
		if _, _, err := Record(s, in, t1); err == nil {
			t.Errorf("%s: expected error", name)
		}
		if len(s.Knowledge) != 0 {
			t.Errorf("%s: store mutated despite error", name)
		}
	}
}

func TestRecordUpdatesInPlace(t *testing.T) {
	s := NewStore()
	if _, _, err := Record(s, rec("storage.format", "fact", "JSON file."), t1); err != nil {
		t.Fatal(err)
	}
	in := rec("storage.format", "decision", "Single JSON file, git-committed.")
	in.Scope = []string{"cmd/", "internal/"} // overlap on none; union expected
	in.Session = ""                          // exercises orDefault
	k, created, err := Record(s, in, t2)
	if err != nil {
		t.Fatal(err)
	}
	if created {
		t.Fatal("expected created=false")
	}
	if k.Kind != "decision" || k.Statement != "Single JSON file, git-committed." {
		t.Fatalf("not updated in place: %+v", k)
	}
	if len(k.Scope) != 2 || k.Scope[0] != "cmd/" || k.Scope[1] != "internal/" {
		t.Fatalf("scope union/sort wrong: %v", k.Scope)
	}
	if len(k.Evidence) != 2 || k.Evidence[1].Session != "unknown" {
		t.Fatalf("evidence: %+v", k.Evidence)
	}
	if !k.Created.Equal(t1) || !k.Updated.Equal(t2) {
		t.Fatalf("timestamps: %+v", k)
	}
}

func TestSupersedeFlow(t *testing.T) {
	s := NewStore()
	if _, _, err := Record(s, rec("extract.pipeline", "decision", "We will build our own extraction pipeline."), t1); err != nil {
		t.Fatal(err)
	}
	in := rec("extract.caller", "decision", "The calling agent performs extraction.")
	in.Supersedes = []string{"extract.pipeline", "extract.pipeline"} // dup target deduped
	if _, _, err := Record(s, in, t2); err != nil {
		t.Fatal(err)
	}

	old := s.Knowledge["extract.pipeline"]
	if old.Status != StatusSuperseded || !old.Updated.Equal(t2) {
		t.Fatalf("target not superseded: %+v", old)
	}
	if len(s.Edges) != 1 || s.Edges[0] != (Edge{Type: EdgeSupersedes, From: "extract.caller", To: "extract.pipeline", At: t2}) {
		t.Fatalf("edges: %+v", s.Edges)
	}

	// Re-recording the successor with the same supersedes must not duplicate the edge.
	in2 := rec("extract.caller", "decision", "The calling agent performs extraction.")
	in2.Supersedes = []string{"extract.pipeline"}
	if _, _, err := Record(s, in2, t2.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if len(s.Edges) != 1 {
		t.Fatalf("duplicate edge added: %+v", s.Edges)
	}
}

func TestRecordOnSupersededKeyErrors(t *testing.T) {
	s := NewStore()
	if _, _, err := Record(s, rec("a", "fact", "old"), t1); err != nil {
		t.Fatal(err)
	}
	in := rec("b", "fact", "new")
	in.Supersedes = []string{"a"}
	if _, _, err := Record(s, in, t2); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Record(s, rec("a", "fact", "resurrected"), t2); err == nil {
		t.Fatal("expected error recording to superseded key")
	}
	if len(s.Knowledge["a"].Evidence) != 1 {
		t.Fatal("superseded entry mutated")
	}
}

func TestSupersedeValidationNoPartialMutation(t *testing.T) {
	s := NewStore()
	in := rec("b", "fact", "new")
	in.Supersedes = []string{"missing"}
	if _, _, err := Record(s, in, t1); err == nil {
		t.Fatal("expected unknown-target error")
	}
	if len(s.Knowledge) != 0 {
		t.Fatalf("partial mutation: %+v", s.Knowledge)
	}

	self := rec("a", "fact", "x")
	self.Supersedes = []string{"a"}
	if _, _, err := Record(s, self, t1); err == nil || !strings.Contains(err.Error(), "itself") {
		t.Fatalf("expected self-supersede error, got %v", err)
	}
}

func TestDispute(t *testing.T) {
	s := NewStore()
	if _, _, err := Record(s, rec("scope.required", "fact", "Every entry needs scope."), t1); err != nil {
		t.Fatal(err)
	}
	k, err := Dispute(s, "scope.required", "reviewer", "Keys suffice for small scopes", t2)
	if err != nil {
		t.Fatal(err)
	}
	if k.Status != StatusDisputed || len(k.Evidence) != 2 || k.Evidence[1].Note == "" {
		t.Fatalf("dispute state: %+v", k)
	}
	if !k.Updated.Equal(t2) {
		t.Fatalf("updated: %+v", k)
	}

	if _, err := Dispute(s, "missing", "s", "", t1); err == nil {
		t.Fatal("expected error for missing key")
	}
}

func TestDisputeSupersededErrors(t *testing.T) {
	s := NewStore()
	if _, _, err := Record(s, rec("a", "fact", "old"), t1); err != nil {
		t.Fatal(err)
	}
	in := rec("b", "fact", "new")
	in.Supersedes = []string{"a"}
	if _, _, err := Record(s, in, t2); err != nil {
		t.Fatal(err)
	}
	if _, err := Dispute(s, "a", "s", "", t2); err == nil || !strings.Contains(err.Error(), "superseded") {
		t.Fatalf("expected superseded error, got %v", err)
	}
}
