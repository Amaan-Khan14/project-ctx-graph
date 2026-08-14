package main

import (
	"github.com/Amaan-Khan14/codedocket/internal/mcp"
	"os"
)

func serveCtx() error {
	return mcp.Serve(os.Stdin, os.Stdout)
}
