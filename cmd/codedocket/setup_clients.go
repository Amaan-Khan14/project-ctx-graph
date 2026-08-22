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
	Name          string // short lowercase name for --clients matching
	Label         string // display name
	GlobalConfig  string // path relative to home dir, "" if unsupported
	ProjectConfig string // path relative to project cwd, "" if unsupported
	GlobalMD      string // global instruction file relative to home, "" if none
	DetectDirs    []string
	merge         func(existing []byte, binPath string) ([]byte, bool, error)
}

var knownClients = []clientDef{
	{
		Name:          "opencode",
		Label:         "opencode",
		GlobalConfig:  ".config/opencode/opencode.json",
		ProjectConfig: "opencode.json",
		GlobalMD:      ".config/opencode/AGENTS.md",
		DetectDirs:    []string{".config/opencode", ".opencode"},
		merge:         mergeOpencodeJSON,
	},
	{
		Name:          "claude",
		Label:         "Claude Code",
		GlobalConfig:  ".claude.json",
		ProjectConfig: ".mcp.json",
		GlobalMD:      ".claude/CLAUDE.md",
		DetectDirs:    []string{".claude", ".claude.json"},
		merge:         mergeMCPServersJSON,
	},
	{
		Name:          "cursor",
		Label:         "Cursor",
		GlobalConfig:  ".cursor/mcp.json",
		ProjectConfig: ".cursor/mcp.json",
		GlobalMD:      "",
		DetectDirs:    []string{".cursor"},
		merge:         mergeMCPServersJSON,
	},
	{
		Name:          "codex",
		Label:         "Codex CLI",
		GlobalConfig:  ".codex/config.toml",
		ProjectConfig: "", // TOML editing stays global-only in V1
		GlobalMD:      ".codex/AGENTS.md",
		DetectDirs:    []string{".codex"},
		merge:         appendCodexTOML,
	},
	{
		// ZCode nests servers under mcp.servers (not the common mcpServers
		// top level); config.json also holds hooks and plugin state, which
		// mergeNestedMap preserves untouched.
		Name:          "zcode",
		Label:         "ZCode",
		GlobalConfig:  ".zcode/cli/config.json",
		ProjectConfig: ".zcode/config.json",
		GlobalMD:      ".zcode/AGENTS.md",
		DetectDirs:    []string{".zcode"},
		merge:         mergeZcodeJSON,
	},
	{
		Name:          "kiro",
		Label:         "Kiro",
		GlobalConfig:  ".kiro/settings/mcp.json",
		ProjectConfig: ".kiro/settings/mcp.json",
		GlobalMD:      "", // steering files are multi-file with frontmatter; out of scope V1
		DetectDirs:    []string{".kiro"},
		merge:         mergeMCPServersJSON,
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
	return mergeNestedMap(existing, []string{"mcp", "codedocket"}, want)
}

// mergeMCPServersJSON upserts the { "mcpServers": { "codedocket": ... } } shape used
// by Claude Code, Cursor, and Kiro.
func mergeMCPServersJSON(existing []byte, binPath string) ([]byte, bool, error) {
	var want map[string]interface{}
	if err := json.Unmarshal([]byte(
		`{"command":"`+binPath+`","args":["serve"]}`), &want); err != nil {
		return nil, false, err // binPath escaping bug; unreachable for sane paths
	}
	return mergeNestedMap(existing, []string{"mcpServers", "codedocket"}, want)
}

// mergeZcodeJSON upserts the { "mcp": { "servers": { "codedocket": ... } } } shape
// used by ZCode's .zcode/cli/config.json.
func mergeZcodeJSON(existing []byte, binPath string) ([]byte, bool, error) {
	var want map[string]interface{}
	if err := json.Unmarshal([]byte(
		`{"command":"`+binPath+`","args":["serve"]}`), &want); err != nil {
		return nil, false, err
	}
	return mergeNestedMap(existing, []string{"mcp", "servers", "codedocket"}, want)
}

// mergeNestedMap upserts root[sections...] = want into arbitrary JSON, creating
// intermediate objects as needed. Idempotent: DeepEqual against the desired
// entry means "Unchanged".
func mergeNestedMap(existing []byte, sections []string, want map[string]interface{}) ([]byte, bool, error) {
	root := map[string]interface{}{}
	if len(bytes.TrimSpace(existing)) > 0 {
		if err := json.Unmarshal(existing, &root); err != nil {
			return nil, false, fmt.Errorf("existing config is not valid JSON (leaving untouched): %w", err)
		}
	}
	cur := root
	for _, sec := range sections[:len(sections)-1] {
		next, _ := cur[sec].(map[string]interface{})
		if next == nil {
			next = map[string]interface{}{}
			cur[sec] = next
		}
		cur = next
	}
	last := sections[len(sections)-1]
	if reflect.DeepEqual(cur[last], want) {
		return existing, false, nil
	}
	cur[last] = want
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
