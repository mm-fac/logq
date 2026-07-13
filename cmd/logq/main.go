// Command logq reads JSON-lines logs and answers questions about them.
//
// v0 implements the `fields` subcommand plus the shared reader and formatter;
// filter/stats/tail are built on the same core in later work items.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mm-fac/logq"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// run is the testable entry point: it takes explicit args and streams and
// returns the process exit code. 0=success, 1=runtime error, 2=usage error.
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	gfs := flag.NewFlagSet("logq", flag.ContinueOnError)
	gfs.SetOutput(stderr)
	format := gfs.String("format", "table", "output format: table|json|logfmt")
	strict := gfs.Bool("strict", false, "treat malformed input lines as a fatal error")
	gfs.Usage = func() { usage(stderr) }
	if err := gfs.Parse(args); err != nil {
		return 2
	}

	rest := gfs.Args()
	if len(rest) == 0 {
		usage(stderr)
		return 2
	}
	cmd, cmdArgs := rest[0], rest[1:]
	switch cmd {
	case "fields":
		return runFields(cmdArgs, *format, *strict, stdin, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "logq: unknown command %q\n", cmd)
		usage(stderr)
		return 2
	}
}

func usage(w io.Writer) {
	fmt.Fprint(w, `usage: logq [--format table|json|logfmt] [--strict] <command> [flags] [files...]

Reads JSON-lines from the given files (concatenated in order) or from stdin.

commands:
  fields    list field keys with their observed value type(s) and record counts

global flags (also accepted after the command):
  --format table|json|logfmt   output format (default table)
  --strict                     exit non-zero on any malformed input line
`)
}

func runFields(args []string, format string, strict bool, stdin io.Reader, stdout, stderr io.Writer) int {
	// Re-register the common flags so they may also appear after the command.
	fs := flag.NewFlagSet("fields", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&format, "format", format, "output format: table|json|logfmt")
	fs.BoolVar(&strict, "strict", strict, "treat malformed input lines as a fatal error")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	f, err := logq.ParseFormat(format)
	if err != nil {
		fmt.Fprintf(stderr, "logq: %v\n", err)
		return 2
	}

	reader, cleanup, err := openInput(fs.Args(), stdin)
	if err != nil {
		fmt.Fprintf(stderr, "logq: %v\n", err)
		return 1
	}
	defer cleanup()

	res, err := logq.Read(reader, strict)
	if err != nil {
		fmt.Fprintf(stderr, "logq: %v\n", err)
		return 1
	}
	if res.Skipped > 0 {
		fmt.Fprintf(stderr, "logq: skipped %d malformed line(s)\n", res.Skipped)
	}

	columns := []string{"field", "types", "count"}
	var rows []*logq.Record
	for _, fi := range logq.Fields(res.Records) {
		rows = append(rows, logq.NewRecord().
			Set("field", fi.Name).
			Set("types", fi.Types).
			Set("count", fi.Count))
	}
	if err := logq.Write(stdout, f, columns, rows); err != nil {
		fmt.Fprintf(stderr, "logq: %v\n", err)
		return 1
	}
	return 0
}

// openInput returns a reader over the concatenated files (a newline is inserted
// between files so a missing trailing newline can't merge two records), or over
// stdin when no files are given. cleanup closes any opened files.
func openInput(files []string, stdin io.Reader) (io.Reader, func(), error) {
	if len(files) == 0 {
		return stdin, func() {}, nil
	}
	var readers []io.Reader
	var closers []io.Closer
	for i, name := range files {
		fh, err := os.Open(name)
		if err != nil {
			for _, c := range closers {
				c.Close()
			}
			return nil, nil, err
		}
		if i > 0 {
			readers = append(readers, strings.NewReader("\n"))
		}
		readers = append(readers, fh)
		closers = append(closers, fh)
	}
	cleanup := func() {
		for _, c := range closers {
			c.Close()
		}
	}
	return io.MultiReader(readers...), cleanup, nil
}
