package mcp

import (
	"bytes"
	"github.com/Amaan-Khan14/codedocket"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestIntegrationWorkflow tests the complete MCP workflow: initialize → record → explore → dispute
func TestIntegrationWorkflow(t *testing.T) {
	tmpDir := t.TempDir()
	ctxDir := filepath.Join(tmpDir, ".codedocket")
	if err := os.MkdirAll(ctxDir, 0755); err != nil {
		t.Fatal(err)
	}
	knowledgePath := filepath.Join(ctxDir, "knowledge.json")
	store := codedocket.NewStore()
	if err := store.Save(knowledgePath); err != nil {
		t.Fatal(err)
	}

	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Simulate MCP session
	var in bytes.Buffer
	var out bytes.Buffer

	// Write a sequence of messages
	messages := []string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","clientInfo":{"name":"integration-test","version":"1.0"}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"codedocket_record","arguments":{"key":"api.versioning","kind":"decision","statement":"Use semantic versioning for all APIs","scope":["api/"]}}}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"codedocket_record","arguments":{"key":"db.migration","kind":"constraint","statement":"All migrations must be reversible","scope":["db/migrations/"]}}}`,
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"codedocket_explore","arguments":{"query":"versioning"}}}`,
		`{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"codedocket_dispute","arguments":{"key":"api.versioning","note":"Should we use calver instead?"}}}`,
		`{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"codedocket_explore","arguments":{"paths":["api/handler.go"]}}}`,
	}

	for _, msg := range messages {
		in.WriteString(msg + "\n")
	}

	// Run the server
	if err := Serve(&in, &out); err != nil {
		t.Fatalf("Serve failed: %v", err)
	}

	// Parse output
	output := out.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	if len(lines) != 6 {
		t.Fatalf("expected 6 responses, got %d", len(lines))
	}

	// Check initialize response
	if !strings.Contains(lines[0], "codedocket") || !strings.Contains(lines[0], "0.1.0") {
		t.Errorf("initialize response invalid: %s", lines[0])
	}

	// Check first record response
	if !strings.Contains(lines[1], "recorded api.versioning") {
		t.Errorf("record response invalid: %s", lines[1])
	}

	// Check second record response
	if !strings.Contains(lines[2], "recorded db.migration") {
		t.Errorf("second record response invalid: %s", lines[2])
	}

	// Check explore response has the versioning entry
	if !strings.Contains(lines[3], "api.versioning") || !strings.Contains(lines[3], "semantic versioning") {
		t.Errorf("explore response missing entry: %s", lines[3])
	}

	// Check dispute response
	if !strings.Contains(lines[4], "disputed api.versioning") {
		t.Errorf("dispute response invalid: %s", lines[4])
	}

	// Check path-scoped explore finds the api entry
	if !strings.Contains(lines[5], "api.versioning") {
		t.Errorf("path-scoped explore should find api.versioning: %s", lines[5])
	}

	// Verify store state
	finalStore, err := codedocket.Load(knowledgePath)
	if err != nil {
		t.Fatal(err)
	}

	if len(finalStore.Knowledge) != 2 {
		t.Errorf("expected 2 knowledge entries, got %d", len(finalStore.Knowledge))
	}

	// Verify disputed status
	results := codedocket.Query(finalStore, codedocket.QueryOpts{Key: "api.versioning"})
	if len(results) != 1 || results[0].Status != "disputed" {
		t.Error("api.versioning should be disputed")
	}
	if len(results[0].Evidence) != 2 {
		t.Errorf("expected 2 evidence entries (record + dispute), got %d", len(results[0].Evidence))
	}
}
