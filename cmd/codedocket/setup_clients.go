package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
)

// clientDef describes one known agent client: how to detect it, where its
// MCP config and global instruction file live, and how to merge our entry.
type clientDef struct {
	Label         string // display name
	GlobalConfig  string // path relative to home dir, "" if unsupported
	ProjectConfig string // path relative to project cwd, "" if unsupported
	GlobalMD      string // global instruction file relative to home, "" if none
	DetectDirs    []string
	merge         func(existing []byte, binPath string) ([]byte, bool, error)
}

var knownClients = []clientDef{
	{
		Label:         "opencode",
		GlobalConfig:  ".config/opencode/opencode.json",
		ProjectConfig: "opencode.json",
		GlobalMD:      ".config/opencode/AGENTS.md",
		DetectDirs:    []string{".config/opencode", ".opencode"},
		merge:         mergeOpencodeJSON,
	},
	{
		Label:         "Claude Code",
		GlobalConfig:  ".claude.json",
		ProjectConfig: ".mcp.json",
		GlobalMD:      ".claude/CLAUDE.md",
		DetectDirs:    []string{".claude", ".claude.json"},
		merge:         mergeMCPServersJSON,
	},
	{
		Label:         "Cursor",
		GlobalConfig:  ".cursor/mcp.json",
		ProjectConfig: ".cursor/mcp.json",
		GlobalMD:      "",
		DetectDirs:    []string{".cursor"},
		merge:         mergeMCPServersJSON,
	},
	{
		Label:         "Codex CLI",
		GlobalConfig:  ".codex/config.toml",
		ProjectConfig: "", // TOML editing stays global-only in V1
		GlobalMD:      ".codex/AGENTS.md",
		DetectDirs:    []string{".codex"},
		merge:         appendCodexTOML,
	},
}

// mergeOpencodeJSON upserts mcp.codedocket into opencode's { "mcp": { ... } } map,
// preserving all unrelated keys. Whole-file rewrite with sorted keys; a
// .bak is made by the caller before writing.
func mergeOpencodeJSON(existing []byte, binPath string) ([]byte, bool, error) {
	var want map[string]interface{}
	if err := json.Unmarshal([]byte(
		`{"type":"local","command":["`+binPath+`","serve"],"enabled":true}`), &want); err != nil {
		return nil, false, err // binPath escaping bug; unreachable for sane paths
	}
	return mergeNestedMap(existing, "mcp", "codedocket", want)
}

// mergeMCPServersJSON upserts the { "mcpServers": { "codedocket": ... } } shape used
// by Claude Code and Cursor.
func mergeMCPServersJSON(existing []byte, binPath string) ([]byte, bool, error) {
	var want map[string]interface{}
	if err := json.Unmarshal([]byte(
		`{"command":"`+binPath+`","args":["serve"]}`), &want); err != nil {
		return nil, false, err
	}
	return mergeNestedMap(existing, "mcpServers", "codedocket", want)
}

// mergeNestedMap upserts root[section][entry] = want into arbitrary JSON.
// Idempotent: DeepEqual against the desired entry means "Unchanged".
func mergeNestedMap(existing []byte, section, entry string, want map[string]interface{}) ([]byte, bool, error) {
	root := map[string]interface{}{}
	if len(bytes.TrimSpace(existing)) > 0 {
		if err := json.Unmarshal(existing, &root); err != nil {
			return nil, false, fmt.Errorf("existing config is not valid JSON (leaving untouched): %w", err)
		}
	}
	sec, _ := root[section].(map[string]interface{})
	if sec == nil {
		sec = map[string]interface{}{}
	}
	if reflect.DeepEqual(sec[entry], want) {
		return existing, false, nil
	}
	sec[entry] = want
	root[section] = sec
	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return nil, false, err
	}
	return append(out, '\n'), true, nil
}

// appendCodexTOML adds a [mcp_servers.codedocket] block if absent. No TOML parser:
// presence detection is the literal section header; anything non-trivial is
// deferred until a real need proves it.
func appendCodexTOML(existing []byte, binPath string) ([]byte, bool, error) {
	if bytes.Contains(existing, []byte("[mcp_servers.codedocket]")) {
		return existing, false, nil
	}
	block := fmt.Sprintf("[mcp_servers.codedocket]\ncommand = %q\nargs = [\"serve\"]\n", binPath)
	trimmed := bytes.TrimRight(existing, "\n")
	if len(bytes.TrimSpace(trimmed)) == 0 {
		return []byte(block), true, nil
	}
	return append(append(trimmed, '\n', '\n'), []byte(block)...), true, nil
}
