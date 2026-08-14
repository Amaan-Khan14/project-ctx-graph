package codedocket

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Load reads knowledge.json from path.
func Load(path string) (*Store, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("no knowledge store at %s (run `codedocket init`)", path)
	}
	if err != nil {
		return nil, fmt.Errorf("reading store: %w", err)
	}
	var s Store
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	if s.Version < 1 || s.Version > CurrentVersion {
		return nil, fmt.Errorf("unsupported store version %d (this build understands up to %d)", s.Version, CurrentVersion)
	}
	if s.Knowledge == nil {
		s.Knowledge = map[string]*Knowledge{}
	}
	if s.Edges == nil {
		s.Edges = []Edge{}
	}
	return &s, nil
}

// Save writes the store atomically: temp file in the target directory, then
// rename. JSON map keys marshal in sorted order, so output is diff-stable
// and git is the history of record.
func (s *Store) Save(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating store dir: %w", err)
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding store: %w", err)
	}
	b = append(b, '\n')

	tmp, err := os.CreateTemp(dir, ".knowledge-*.tmp")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return fmt.Errorf("writing temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return fmt.Errorf("closing temp file: %w", err)
	}
	_ = os.Chmod(tmp.Name(), 0o644)
	if err := os.Rename(tmp.Name(), path); err != nil {
		os.Remove(tmp.Name())
		return fmt.Errorf("replacing store: %w", err)
	}
	return nil
}

// ErrNoStore is returned by ResolveStore when no knowledge store is found
// walking up from the working directory.
var ErrNoStore = errors.New("no codedocket store found")

// ResolveStore walks up from the current directory (git-style) looking for
// .codedocket/knowledge.json, falling back to legacy .ctx/knowledge.json for
// pre-rename projects. The single resolver shared by the CLI and the MCP
// server. Note: `codedocket init` does NOT walk up — it creates in cwd only.
func ResolveStore() (storeDir, knowledgePath string, err error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", "", fmt.Errorf("getting current directory: %w", err)
	}
	for {
		for _, name := range []string{".codedocket", ".ctx"} {
			storeDir = filepath.Join(dir, name)
			knowledgePath = filepath.Join(storeDir, "knowledge.json")

			info, err := os.Stat(knowledgePath)
			if err == nil && !info.IsDir() {
				return storeDir, knowledgePath, nil
			}
			if err != nil && !errors.Is(err, fs.ErrNotExist) {
				return "", "", fmt.Errorf("checking %s: %w", knowledgePath, err)
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", "", ErrNoStore
		}
		dir = parent
	}
}
