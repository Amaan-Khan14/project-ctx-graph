package codedocket

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveStore(t *testing.T) {
	// Create temporary test tree: /tmp/x/a/b/c with store at /tmp/x/.codedocket
	tmpRoot := t.TempDir()
	storeRoot := filepath.Join(tmpRoot, "x")
	deepDir := filepath.Join(storeRoot, "a", "b", "c")

	if err := os.MkdirAll(deepDir, 0755); err != nil {
		t.Fatalf("creating test dirs: %v", err)
	}

	ctxDir := filepath.Join(storeRoot, ".codedocket")
	if err := os.MkdirAll(ctxDir, 0755); err != nil {
		t.Fatalf("creating .codedocket dir: %v", err)
	}

	knowledgePath := filepath.Join(ctxDir, "knowledge.json")
	store := NewStore()
	if err := store.Save(knowledgePath); err != nil {
		t.Fatalf("creating test store: %v", err)
	}

	// Test 1: resolve from deep subdirectory should find the store
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("saving original dir: %v", err)
	}
	defer os.Chdir(origDir)

	if err := os.Chdir(deepDir); err != nil {
		t.Fatalf("changing to test dir: %v", err)
	}

	gotCtxDir, gotPath, err := ResolveStore()
	if err != nil {
		t.Fatalf("ResolveStore from %s: %v", deepDir, err)
	}

	// Normalize paths for comparison (macOS has /var -> /private/var symlink)
	wantCtxDir, _ := filepath.EvalSymlinks(ctxDir)
	gotCtxDirNorm, _ := filepath.EvalSymlinks(gotCtxDir)
	wantPath, _ := filepath.EvalSymlinks(knowledgePath)
	gotPathNorm, _ := filepath.EvalSymlinks(gotPath)

	if gotCtxDirNorm != wantCtxDir {
		t.Errorf("ctxDir: got %q, want %q", gotCtxDirNorm, wantCtxDir)
	}
	if gotPathNorm != wantPath {
		t.Errorf("knowledgePath: got %q, want %q", gotPathNorm, wantPath)
	}

	// Test 2: resolve from a tree without any store should return ErrNoStore
	noStoreRoot := filepath.Join(tmpRoot, "no-store-tree")
	if err := os.MkdirAll(noStoreRoot, 0755); err != nil {
		t.Fatalf("creating no-store tree: %v", err)
	}

	if err := os.Chdir(noStoreRoot); err != nil {
		t.Fatalf("changing to no-store dir: %v", err)
	}

	_, _, err = ResolveStore()
	if !errors.Is(err, ErrNoStore) {
		t.Errorf("ResolveStore from store-less tree: got error %v, want ErrNoStore", err)
	}
}

func TestResolveStoreLegacyCtxFallback(t *testing.T) {
	tmpRoot := t.TempDir()
	storeRoot := filepath.Join(tmpRoot, "legacy")
	deepDir := filepath.Join(storeRoot, "a", "b")
	if err := os.MkdirAll(deepDir, 0755); err != nil {
		t.Fatalf("creating test dirs: %v", err)
	}

	legacyDir := filepath.Join(storeRoot, ".ctx")
	knowledgePath := filepath.Join(legacyDir, "knowledge.json")
	store := NewStore()
	if err := store.Save(knowledgePath); err != nil {
		t.Fatalf("creating legacy store: %v", err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("saving original dir: %v", err)
	}
	defer os.Chdir(origDir)

	if err := os.Chdir(deepDir); err != nil {
		t.Fatalf("changing to test dir: %v", err)
	}

	gotDir, gotPath, err := ResolveStore()
	if err != nil {
		t.Fatalf("ResolveStore from legacy tree: %v", err)
	}

	wantDir, _ := filepath.EvalSymlinks(legacyDir)
	gotDirNorm, _ := filepath.EvalSymlinks(gotDir)
	wantPath, _ := filepath.EvalSymlinks(knowledgePath)
	gotPathNorm, _ := filepath.EvalSymlinks(gotPath)

	if gotDirNorm != wantDir {
		t.Errorf("store dir: got %q, want %q", gotDirNorm, wantDir)
	}
	if gotPathNorm != wantPath {
		t.Errorf("knowledge path: got %q, want %q", gotPathNorm, wantPath)
	}
}
