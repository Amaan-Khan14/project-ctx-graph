package projectcontext

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
		return nil, fmt.Errorf("no knowledge store at %s (run `projectcontext init`)", path)
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
