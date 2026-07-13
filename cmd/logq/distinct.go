package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/mm-fac/logq"
)

// runDistinct implements `logq distinct <field> [files...]`: one row per
// distinct value of the top-level field, with its occurrence count. Values of
// different JSON types stay distinct even when they render alike (1 vs "1"),
// and rows are ordered by each value's canonical JSON rendering. Records
// missing the field are skipped and their count is reported on stderr via the
// shared skipped-count message. Output honors --format and the same
// malformed-line/--strict contract as the other subcommands.
func runDistinct(args []string, format string, strict bool, stdin io.Reader, stdout, stderr io.Writer) int {
	// Re-register the common flags so they may also appear after the command.
	fs := flag.NewFlagSet("distinct", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&format, "format", format, "output format: table|json|logfmt")
	fs.BoolVar(&strict, "strict", strict, "treat malformed input lines as a fatal error")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	rest := fs.Args()
	if len(rest) == 0 {
		fmt.Fprintln(stderr, "logq: distinct requires a <field> argument")
		distinctUsage(stderr)
		return 2
	}
	// The first positional is the field; any remaining positionals are input
	// files (global input handling).
	field, files := rest[0], rest[1:]

	f, err := logq.ParseFormat(format)
	if err != nil {
		fmt.Fprintf(stderr, "logq: %v\n", err)
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

	values, missing := logq.Distinct(res.Records, field)
	if missing > 0 {
		fmt.Fprintf(stderr, "logq: skipped %d record(s) missing field %q\n", missing, field)
	}

	columns := []string{"value", "count"}
	var rows []*logq.Record
	for _, dv := range values {
		rows = append(rows, logq.NewRecord().
			Set("value", dv.Value).
			Set("count", dv.Count))
	}
	if err := logq.Write(stdout, f, columns, rows); err != nil {
		fmt.Fprintf(stderr, "logq: %v\n", err)
		return 1
	}
	return 0
}

func distinctUsage(w io.Writer) {
	fmt.Fprint(w, `usage: logq distinct <field> [files...]

List each distinct value of the top-level <field> with its occurrence count,
one row per value. Values of different JSON types are distinct even when they
render alike (1 vs "1"). Rows are sorted by the value's canonical JSON
rendering. Records missing the field are skipped and counted on stderr.
`)
}
