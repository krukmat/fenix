// Command fenixlsp is the Carta DSL Language Server. // CLSF-40
//
// Usage:
//
//	fenixlsp --stdio
//
// The --stdio flag (required) connects the LSP server to stdin/stdout so that
// editors can launch it as a child process.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/matiasleandrokruk/fenix/internal/lsp"
	"github.com/matiasleandrokruk/fenix/internal/lsp/handlers"
)

func main() {
	stdio := flag.Bool("stdio", false, "run LSP server over stdin/stdout")
	flag.Parse()

	if !*stdio {
		fmt.Fprintln(os.Stderr, "fenixlsp: --stdio flag is required")
		os.Exit(1)
	}

	srv := lsp.NewServer(os.Stdin, os.Stdout)
	srv.WithDiagnostics(handlers.NewDiagnosticsHandler(srv.Docs())) // CLSF-42
	srv.WithCompletion(handlers.NewCompletionHandler(srv.Docs()))   // CLSF-43
	srv.WithHover(handlers.NewHoverHandler(srv.Docs()))             // CLSF-44
	if err := srv.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "fenixlsp: %v\n", err)
		os.Exit(1)
	}
}
