package main

import (
	"os"
	"projectcontext/internal/mcp"
)

func serveCtx() error {
	return mcp.Serve(os.Stdin, os.Stdout)
}
