package main

import (
	"flag"
	"fmt"
	"projectcontext"
	"strings"
	"time"
)

func recordCtx(args []string) error {
	fs := flag.NewFlagSet("record", flag.ExitOnError)

	key := fs.String("key", "", "stable key")
	kind := fs.String("kind", "", "decision kind")
	statement := fs.String("statement", "", "statement")
	scope := fs.String("scope", "", "comma-separated scopes")
	supersedes := fs.String("supersedes", "", "comma-separated keys")
	session := fs.String("session", "cli", "session id")
	note := fs.String("note", "", "optional note")

	fs.Usage = func() {
		fmt.Println("record --key <key> --kind <kind> --statement <statement> --scope <scope> [--supersedes <k1,k2>] [--session <session>] [--note <text>]")
	}

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("error parsing flags: %w", err)
	}

	if *key == "" {
		return fmt.Errorf("key is required")
	}
	if *kind == "" {
		return fmt.Errorf("kind is required")
	}
	if *statement == "" {
		return fmt.Errorf("statement is required")
	}
	if *scope == "" {
		return fmt.Errorf("scope is required")
	}

	_, knowledgeFilePath, err := getDir()
	if err != nil {
		return err
	}

	store, err := projectcontext.Load(knowledgeFilePath)
	if err != nil {
		return err
	}

	in := projectcontext.RecordInput{
		Key:        *key,
		Kind:       *kind,
		Statement:  *statement,
		Scope:      csv(*scope),
		Supersedes: csv(*supersedes),
		Session:    *session,
		Note:       *note,
	}

	if _, _, err := projectcontext.Record(store, in, time.Now()); err != nil {
		return err
	}

	return store.Save(knowledgeFilePath)
}

func csv(s string) []string {
	if s == "" {
		return nil
	}

	parts := strings.Split(s, ",")

	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}

	return out
}
