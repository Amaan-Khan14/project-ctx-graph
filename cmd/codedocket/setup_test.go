package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMergeOpencodeJSON(t *testing.T) {
	bin := "/home/u/.local/bin/codedocket"

	// empty file
	merged, changed, err := mergeOpencodeJSON(nil, bin)
	if err != nil || !changed {
		t.Fatalf("empty: changed=%v err=%v", changed, err)
	}
	var root map[string]interface{}
	if err := json.Unmarshal(merged, &root); err != nil {
		t.Fatal(err)
	}
	mcp := root["mcp"].(map[string]interface{})
	entry := mcp["codedocket"].(map[string]interface{})
	if entry["type"] != "local" || entry["enabled"] != true {
		t.Fatalf("entry: %+v", entry)
	}
	cmd := entry["command"].([]interface{})
	if cmd[0] != bin || cmd[1] != "serve" {
		t.Fatalf("command: %v", cmd)
	}

	// preserving unrelated keys
	existing := []byte(`{"model": "x", "mcp": {"codegraph": {"type":"local","command":["codegraph","serve","--mcp"],"enabled":true}}}`)
	merged2, changed2, err := mergeOpencodeJSON(existing, bin)
	if err != nil || !changed2 {
		t.Fatalf("preserve: changed=%v err=%v", changed2, err)
	}
	var root2 map[string]interface{}
	json.Unmarshal(merged2, &root2)
	if root2["model"] != "x" {
		t.Fatal("unrelated top-level key lost")
	}
	mcp2 := root2["mcp"].(map[string]interface{})
	if _, ok := mcp2["codegraph"]; !ok {
		t.Fatal("existing sibling server lost")
	}
	if _, ok := mcp2["codedocket"]; !ok {
		t.Fatal("codedocket not added")
	}

	// idempotent: merging the merged output reports unchanged
	merged3, changed3, err := mergeOpencodeJSON(merged2, bin)
	if err != nil {
		t.Fatal(err)
	}
	if changed3 {
		t.Fatalf("second merge should be Unchanged")
	}
	if string(merged3) != string(merged2) {
		t.Fatal("unchanged result differs from input")
	}

	// invalid JSON surfaces, nothing written
	if _, _, err := mergeOpencodeJSON([]byte(`{bad`), bin); err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}

func TestMergeMCPServersJSON(t *testing.T) {
	bin := "/x/codedocket"
	merged, changed, err := mergeMCPServersJSON(nil, bin)
	if err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	var root map[string]interface{}
	json.Unmarshal(merged, &root)
	servers := root["mcpServers"].(map[string]interface{})
	entry := servers["codedocket"].(map[string]interface{})
	if entry["command"] != bin || entry["args"].([]interface{})[0] != "serve" {
		t.Fatalf("entry: %+v", entry)
	}

	if _, changed, _ := mergeMCPServersJSON(merged, bin); changed {
		t.Fatal("second merge should be Unchanged")
	}
}

func TestAppendCodexTOML(t *testing.T) {
	bin := "/x/codedocket"

	out, changed, err := appendCodexTOML(nil, bin)
	if err != nil || !changed {
		t.Fatalf("empty: changed=%v err=%v", changed, err)
	}
	if !strings.HasPrefix(string(out), "[mcp_servers.codedocket]") {
		t.Fatalf("block placement:\n%s", out)
	}

	existing := []byte("[profiles.default]\nname = \"work\"\n")
	out2, changed2, _ := appendCodexTOML(existing, bin)
	if !changed2 {
		t.Fatal("expected change")
	}
	if !strings.Contains(string(out2), "[profiles.default]") || !strings.Contains(string(out2), "[mcp_servers.codedocket]") {
		t.Fatalf("existing TOML not preserved:\n%s", out2)
	}

	if _, changed3, _ := appendCodexTOML(out2, "/other/path/codedocket"); changed3 {
		t.Fatal("second append must be no-op regardless of binPath")
	}
}

func TestClientDetected(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if clientDetected(home, knownClients[0]) {
		t.Fatal("opencode should not be detected in empty home")
	}
	mkdir(t, home, ".config/opencode")
	if !clientDetected(home, knownClients[0]) {
		t.Fatal("opencode should be detected once .config/opencode exists")
	}
}

func TestSelectClientsByName(t *testing.T) {
	got, err := selectClients(t.TempDir(), "opencode,claude", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Label != "opencode" || got[1].Label != "Claude Code" {
		t.Fatalf("selection: %+v", got)
	}
	if _, err := selectClients(t.TempDir(), "nope", false); err == nil {
		t.Fatal("unknown client must error")
	}
}

func mkdir(t *testing.T, root, rel string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, rel), 0o755); err != nil {
		t.Fatal(err)
	}
}
