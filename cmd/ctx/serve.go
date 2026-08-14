package main

import (
	"github.com/Amaan-Khan14/project-ctx-graph/internal/mcp"
	"os"
)

func serveCtx() error {
	return mcp.Serve(os.Stdin, os.Stdout)
}
