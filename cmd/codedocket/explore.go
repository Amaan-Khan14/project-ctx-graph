package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/Amaan-Khan14/codedocket"
	"strings"
)

func exploreCtx(args []string) error {
	fs := flag.NewFlagSet("explore", flag.ExitOnError)

	query := fs.String("query", "", "text to search for")
	path := fs.String("path", "", "comma-separated paths")
	kind := fs.String("kind", "", "knowledge kind")
	key := fs.String("key", "", "knowledge key")
	includeSuperseded := fs.Bool("include-superseded", false, "include superseded knowledge")
	jsonOutput := fs.Bool("json", false, "output results as JSON")

	fs.Usage = func() {
		fmt.Println("explore [--query <text>] [--path <p1,p2>] [--kind <kind>] [--key <key>] [--include-superseded] [--json]")
	}

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("error parsing flags: %w", err)
	}

	knowledgeFilePath, err := mustStore()
	if err != nil {
		return err
	}

	store, err := codedocket.Load(knowledgeFilePath)
	if err != nil {
		return fmt.Errorf("error loading knowledge.json: %w", err)
	}

	in := codedocket.QueryOpts{
		Text:              *query,
		Paths:             csv(*path),
		Kind:              *kind,
		Key:               *key,
		IncludeSuperseded: *includeSuperseded,
	}

	results := codedocket.Query(store, in)

	if *jsonOutput {
		data, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return fmt.Errorf("error rendering JSON: %w", err)
		}

		fmt.Println(string(data))
		return nil
	}

	if len(results) == 0 {
		fmt.Println("no matching knowledge")
		return nil
	}

	fmt.Print(renderHuman(results))

	return nil
}

func renderHuman(knowledge []*codedocket.Knowledge) string {
	var out strings.Builder

	for i, k := range knowledge {
		status := string(k.Status)

		fmt.Fprintf(
			&out,
			"%s  [%s",
			k.Key,
			k.Kind,
		)

		if status != "active" {
			fmt.Fprintf(&out, ", %s", strings.ToUpper(status))
		}

		fmt.Fprintf(
			&out,
			"]  (evidence: %d)\n",
			len(k.Evidence),
		)

		fmt.Fprintf(&out, "  %s\n", k.Statement)

		fmt.Fprintf(
			&out,
			"  scope: %s  updated: %s\n",
			strings.Join(k.Scope, ", "),
			k.Updated.Format("2006-01-02"),
		)

		if i < len(knowledge)-1 {
			out.WriteString("\n")
		}
	}

	return out.String()
}
