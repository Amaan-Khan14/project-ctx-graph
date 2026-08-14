package mcp

import (
	"encoding/json"
	"github.com/Amaan-Khan14/codedocket"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHandleMessage(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantNil  bool // true for notifications
		contains []string
	}{
		{
			name:     "initialize echoes version",
			input:    `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","clientInfo":{"name":"test-client","version":"1.0"}}}`,
			contains: []string{`"protocolVersion":"2025-03-26"`, `"name":"codedocket"`, `"version":"0.1.0"`},
		},
		{
			name:    "notification returns nil",
			input:   `{"jsonrpc":"2.0","method":"notifications/initialized"}`,
			wantNil: true,
		},
		{
			name:     "tools/list returns 3 tools",
			input:    `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
			contains: []string{`"codedocket_explore"`, `"codedocket_record"`, `"codedocket_dispute"`, `"inputSchema"`},
		},
		{
			name:     "unknown method returns error",
			input:    `{"jsonrpc":"2.0","id":3,"method":"unknown"}`,
			contains: []string{`"error"`, `"code":-32601`, `method not found`},
		},
		{
			name:     "garbage returns parse error",
			input:    `{invalid json`,
			contains: []string{`"error"`, `"code":-32700`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := handleMessage([]byte(tt.input))

			if tt.wantNil {
				if response != nil {
					t.Errorf("expected nil response for notification, got %s", response)
				}
				return
			}

			if response == nil {
				t.Fatal("expected response, got nil")
			}

			respStr := string(response)
			for _, want := range tt.contains {
				if !strings.Contains(respStr, want) {
					t.Errorf("response missing %q\ngot: %s", want, respStr)
				}
			}
		})
	}
}

func TestToolCalls(t *testing.T) {
	// Set up temp store
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

	// Change to temp dir so resolveStore finds it
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Initialize client
	initMsg := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","clientInfo":{"name":"test-agent","version":"1.0"}}}`
	handleMessage([]byte(initMsg))

	t.Run("record creates entry", func(t *testing.T) {
		msg := `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"codedocket_record","arguments":{"key":"test.key","kind":"decision","statement":"Test statement","scope":["."]}}}`
		resp := handleMessage([]byte(msg))

		if !strings.Contains(string(resp), "recorded test.key") {
			t.Errorf("expected success message, got: %s", resp)
		}

		// Verify file was written
		store, err := codedocket.Load(knowledgePath)
		if err != nil {
			t.Fatal(err)
		}
		if len(store.Knowledge) != 1 {
			t.Errorf("expected 1 entry, got %d", len(store.Knowledge))
		}
	})

	t.Run("explore returns recorded entry", func(t *testing.T) {
		msg := `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"codedocket_explore","arguments":{"key":"test.key"}}}`
		resp := handleMessage([]byte(msg))

		if !strings.Contains(string(resp), "test.key") || !strings.Contains(string(resp), "Test statement") {
			t.Errorf("expected entry in results, got: %s", resp)
		}

		// Verify it's valid JSON array
		var parsed struct {
			Result struct {
				Content []struct {
					Text string `json:"text"`
				} `json:"content"`
			} `json:"result"`
		}
		if err := json.Unmarshal(resp, &parsed); err != nil {
			t.Fatalf("invalid JSON response: %v", err)
		}

		if len(parsed.Result.Content) == 0 {
			t.Error("expected content in response")
		}
	})

	t.Run("record with invalid kind returns error", func(t *testing.T) {
		msg := `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"codedocket_record","arguments":{"key":"bad.key","kind":"invalid","statement":"Test","scope":["."]}}}`
		resp := handleMessage([]byte(msg))

		if !strings.Contains(string(resp), "isError") {
			t.Errorf("expected isError in response, got: %s", resp)
		}
	})

	t.Run("unknown tool returns error", func(t *testing.T) {
		msg := `{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"unknown_tool","arguments":{}}}`
		resp := handleMessage([]byte(msg))

		if !strings.Contains(string(resp), "unknown tool") {
			t.Errorf("expected unknown tool error, got: %s", resp)
		}
	})
}

func TestNoStore(t *testing.T) {
	// Run from a directory without .codedocket
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	msg := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"codedocket_explore","arguments":{}}}`
	resp := handleMessage([]byte(msg))

	if !strings.Contains(string(resp), "no .codedocket store found") || !strings.Contains(string(resp), "codedocket init") {
		t.Errorf("expected no store error with hint, got: %s", resp)
	}
}
