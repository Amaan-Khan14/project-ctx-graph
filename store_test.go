package projectcontext

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSaveLoadRoundtrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".projectcontext", "knowledge.json")
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)

	s := NewStore()
	s.Knowledge["merge.rules"] = &Knowledge{
		Key:       "merge.rules",
		Kind:      "decision",
		Statement: "Merge is deterministic.",
		Scope:     []string{"internal/"},
		Status:    StatusActive,
		Evidence:  []Evidence{{Session: "s1", At: now}},
		Created:   now,
		Updated:   now,
	}
	s.Edges = append(s.Edges, Edge{Type: EdgeSupersedes, From: "b", To: "a", At: now})

	if err := s.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	k := got.Knowledge["merge.rules"]
	if k == nil || k.Statement != "Merge is deterministic." || k.Status != StatusActive || len(k.Evidence) != 1 {
		t.Fatalf("roundtrip mismatch: %+v", k)
	}
	if len(got.Edges) != 1 || got.Edges[0].From != "b" || got.Version != CurrentVersion {
		t.Fatalf("edges/version mismatch: %+v", got)
	}

	// Saving again must be byte-identical: diff-stable for git.
	b1, _ := os.ReadFile(path)
	if err := got.Save(path); err != nil {
		t.Fatalf("Save 2: %v", err)
	}
	b2, _ := os.ReadFile(path)
	if string(b1) != string(b2) {
		t.Fatalf("save not deterministic\n--- first ---\n%s\n--- second ---\n%s", b1, b2)
	}

	// No temp files left behind.
	entries, _ := os.ReadDir(filepath.Dir(path))
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".knowledge-") {
			t.Fatalf("temp file left behind: %s", e.Name())
		}
	}
}

func TestLoadMissing(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "nope.json"))
	if err == nil || !strings.Contains(err.Error(), "init") {
		t.Fatalf("expected init hint, got %v", err)
	}
}

func TestLoadUnsupportedVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "knowledge.json")
	if err := os.WriteFile(path, []byte(`{"version": 99, "knowledge": {}, "edges": []}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil || !strings.Contains(err.Error(), "version") {
		t.Fatalf("expected version error, got %v", err)
	}
}
