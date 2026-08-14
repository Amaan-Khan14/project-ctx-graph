package main

import (
	"flag"
	"fmt"
	"github.com/Amaan-Khan14/project-ctx-graph"
	"time"
)

func disputeCtx(args []string) error {
	fs := flag.NewFlagSet("dispute", flag.ExitOnError)

	key := fs.String("key", "", "stable key")
	note := fs.String("note", "", "optional note")
	session := fs.String("session", "cli", "session id")

	fs.Usage = func() {
		fmt.Println("dispute --key <key> [--note <text>] [--session <session>]")
	}

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("error parsing flags: %w", err)
	}

	if *key == "" {
		return fmt.Errorf("key is required")
	}

	knowledgeFilePath, err := mustStore()
	if err != nil {
		return err
	}

	store, err := projectcontext.Load(knowledgeFilePath)
	if err != nil {
		return err
	}

	if _, err := projectcontext.Dispute(
		store,
		*key,
		*session,
		*note,
		time.Now(),
	); err != nil {
		return err
	}

	if err := store.Save(knowledgeFilePath); err != nil {
		return err
	}

	fmt.Printf("disputed %s\n", *key)

	return nil
}
