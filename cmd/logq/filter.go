package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/mm-fac/logq"
)

// runFilter implements `logq filter <predicate>... [files...]`. Positional
// arguments containing a comparison operator are parsed as predicates; the rest
// are treated as input files (global input handling). Matched records are
// printed in full, honoring --format and the same malformed-line/--strict
// contract as `fields`.
func runFilter(args []string, format string, strict bool, stdin io.Reader, stdout, stderr io.Writer) int {
	// Re-register the common flags so they may also appear after the command.
	fs := flag.NewFlagSet("filter", flag.ContinueOnError)
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

	// Split positional args into predicates and input files.
	var preds []logq.Predicate
	var files []string
	for _, a := range fs.Args() {
		if logq.HasOperator(a) {
			p, err := logq.ParsePredicate(a)
			if err != nil {
				fmt.Fprintf(stderr, "logq: %v\n", err)
				filterUsage(stderr)
				return 2
			}
			preds = append(preds, p)
			continue
		}
		files = append(files, a)
	}
	if len(preds) == 0 {
		fmt.Fprintf(stderr, "logq: filter requires at least one predicate\n")
		filterUsage(stderr)
		return 2
	}

	reader, cleanup, err := openInput(files, stdin)
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

	matched := logq.Filter(res.Records, preds)
	if err := logq.Write(stdout, f, filterColumns(matched), matched); err != nil {
		fmt.Fprintf(stderr, "logq: %v\n", err)
		return 1
	}
	return 0
}

// filterColumns is the union of keys across the matched records, in first-seen
// order, so the formatter renders every field the surviving records carry.
func filterColumns(records []*logq.Record) []string {
	seen := make(map[string]bool)
	var cols []string
	for _, rec := range records {
		for _, k := range rec.Keys() {
			if !seen[k] {
				seen[k] = true
				cols = append(cols, k)
			}
		}
	}
	return cols
}

func filterUsage(w io.Writer) {
	fmt.Fprint(w, `usage: logq filter <predicate>... [files...]

A predicate is `+"`field OP value`"+` with OP one of: == != > >= < <= ~
Multiple predicates are ANDed. A number literal compares numerically,
otherwise the comparison is on the value's string form; ~ is substring contains.
`)
}
