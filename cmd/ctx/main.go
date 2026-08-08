package main

import (
	"fmt"
	"os"
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

	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: ctx <command> [args...]")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "commands:")
	fmt.Fprintln(os.Stderr, "  init      initialize a new knowledge store")
	fmt.Fprintln(os.Stderr, "  record    record a decision or observation")
	fmt.Fprintln(os.Stderr, "  dispute   mark knowledge as disputed")
}
