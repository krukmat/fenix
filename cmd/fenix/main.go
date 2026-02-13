// FenixCRM - Agentic CRM OS
// Task 1.1: Project Setup - Entry point
// Following implementation plan exactly

package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/matiasleandrokruk/fenix/internal/version"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout))
}

func run(args []string, out io.Writer) int {
	fs := flag.NewFlagSet("fenix", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	showVersion := fs.Bool("version", false, "Show version information")
	showHelp := fs.Bool("help", false, "Show help")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *showVersion {
		fmt.Fprintln(out, version.String()) //nolint:errcheck
		return 0
	}

	if *showHelp {
		printHelp(out)
		return 0
	}

	// Default: print version (as per test requirement)
	fmt.Fprintln(out, version.String()) //nolint:errcheck
	return 0
}

func printHelp(out io.Writer) {
	helpText := `FenixCRM - Agentic CRM OS

Usage:
  fenix [options]

Options:
  --version    Show version information
  --help       Show this help message

Commands:
  serve        Start the server (default)
  migrate      Run database migrations

Examples:
  fenix --version
  fenix serve --port 8080
  fenix migrate up`
	fmt.Fprintln(out, helpText) //nolint:errcheck
}
