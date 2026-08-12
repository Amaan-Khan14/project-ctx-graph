package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"projectcontext"
)

func initCtx(args []string) error {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	force := fs.Bool("force", false, "overwrite existing store")

	if err := fs.Parse(args); err != nil {
		return err
	}

	ctxDir, knowledgeFilePath, err := getDir()
	if err != nil {
		return err
	}

	if _, err := os.Stat(knowledgeFilePath); err == nil {
		if !*force {
			return fmt.Errorf("knowledge store already exists at %s (use --force to overwrite)", knowledgeFilePath)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("checking store: %w", err)
	}

	store := projectcontext.NewStore()

	if err := store.Save(knowledgeFilePath); err != nil {
		return fmt.Errorf("initializing knowledge store: %w", err)
	}

	fmt.Printf("initialized ctx at %s\n", ctxDir)
	return nil

}

// getDir resolves .ctx in the CURRENT directory only. init intentionally
// never walks up — init creates; projectcontext.ResolveStore discovers.
func getDir() (string, string, error) {
	pwdDir, err := os.Getwd()
	if err != nil {
		return "", "", fmt.Errorf("getting current directory: %w", err)
	}
	ctxDir := filepath.Join(pwdDir, ".ctx")
	return ctxDir, filepath.Join(ctxDir, "knowledge.json"), nil
}
