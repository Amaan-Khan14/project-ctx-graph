package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureAgentsSnippetCreates(t *testing.T) {
	dir := t.TempDir()

	outcome, err := ensureAgentsSnippet(dir)
	if err != nil {
		t.Fatalf("ensureAgentsSnippet: %v", err)
	}
	if outcome != "created" {
		t.Fatalf("outcome = %q, want created", outcome)
	}

	data, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	for _, want := range []string{markerBegin, markerEnd, "ctx_explore", "ctx_record", "knowledge.json", "supersedes"} {
		if !strings.Contains(s, want) {
			t.Errorf("snippet missing %q", want)
		}
	}
}

func TestEnsureAgentsSnippetAppendsAndPreserves(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "AGENTS.md")
	original := "# Existing team notes\n\n- use prettier\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	outcome, err := ensureAgentsSnippet(dir)
	if err != nil {
		t.Fatal(err)
	}
	if outcome != "appended" {
		t.Fatalf("outcome = %q, want appended", outcome)
	}

	data, _ := os.ReadFile(path)
	s := string(data)
	if !strings.HasPrefix(s, original) {
		t.Errorf("existing content not preserved:\n%s", s)
	}
	if !strings.Contains(s, markerBegin) {
		t.Error("snippet not appended")
	}
}

func TestEnsureAgentsSnippetIdempotent(t *testing.T) {
	dir := t.TempDir()

	if _, err := ensureAgentsSnippet(dir); err != nil {
		t.Fatal(err)
	}
	first, _ := os.ReadFile(filepath.Join(dir, "AGENTS.md"))

	outcome, err := ensureAgentsSnippet(dir)
	if err != nil {
		t.Fatal(err)
	}
	if outcome != "present" {
		t.Fatalf("second run outcome = %q, want present", outcome)
	}
	second, _ := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if string(first) != string(second) {
		t.Fatal("second run modified the file")
	}
}

func TestEnsureAgentsSnippetEmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "AGENTS.md")
	if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := ensureAgentsSnippet(dir); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if !strings.HasPrefix(string(data), markerBegin) {
		t.Errorf("empty file should become snippet-only, got prefix:\n%q", string(data)[:80])
	}
}
