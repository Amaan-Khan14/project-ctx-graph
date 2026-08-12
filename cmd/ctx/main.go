package main

import (
	"errors"
	"fmt"
	"os"
	"projectcontext"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {

	case "init":

		if err := initCtx(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "ctx init:", err)
			os.Exit(1)
		}

	case "record":
		if err := recordCtx(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "ctx record:", err)
			os.Exit(1)
		}

	case "dispute":
		if err := disputeCtx(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "ctx dispute:", err)
			os.Exit(1)
		}

	case "explore":
		if err := exploreCtx(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "ctx explore:", err)
			os.Exit(1)
		}

	case "serve":
		if err := serveCtx(); err != nil {
			fmt.Fprintln(os.Stderr, "ctx serve:", err)
			os.Exit(1)
		}

	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}

// mustStore resolves the nearest .ctx store walking up from cwd, with a
// friendly error when none exists.
func mustStore() (string, error) {
	_, knowledgePath, err := projectcontext.ResolveStore()
	if errors.Is(err, projectcontext.ErrNoStore) {
		return "", fmt.Errorf("no .ctx store found walking up from cwd; run `ctx init`")
	}
	return knowledgePath, err
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: ctx <command> [args...]")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "commands:")
	fmt.Fprintln(os.Stderr, "  init      initialize a new knowledge store")
	fmt.Fprintln(os.Stderr, "  record    record a decision or observation")
	fmt.Fprintln(os.Stderr, "  dispute   mark knowledge as disputed")
	fmt.Fprintln(os.Stderr, "  explore   explore project knowledge")
	fmt.Fprintln(os.Stderr, "  serve     run MCP stdio server")
}
