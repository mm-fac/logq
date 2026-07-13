package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/mm-fac/logq"
)

// runSort implements `logq sort --by <field> [--desc] [files...]`: it prints all
// records ordered by the top-level field. Two JSON numbers compare numerically;
// any other pair compares by canonical JSON rendering, bytewise. Records missing
// the field sort last (keeping input order among themselves) regardless of
// --desc, and the sort is stable so ties keep input order. Output honors
// --format and the same malformed-line/--strict contract as the other
// subcommands. --by is required.
func runSort(args []string, format string, strict bool, stdin io.Reader, stdout, stderr io.Writer) int {
	// Re-register the common flags so they may also appear after the command.
	fs := flag.NewFlagSet("sort", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&format, "format", format, "output format: table|json|logfmt")
	fs.BoolVar(&strict, "strict", strict, "treat malformed input lines as a fatal error")
	by := fs.String("by", "", "top-level field to sort by (required)")
	desc := fs.Bool("desc", false, "sort in descending order")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *by == "" {
		fmt.Fprintln(stderr, "logq: sort requires --by <field>")
		sortUsage(stderr)
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

	rows := logq.Sort(res.Records, *by, *desc)
	// Columns are the union of keys across the printed records, in first-seen
	// order — the same ordering fields/tail use.
	var columns []string
	for _, fi := range logq.Fields(rows) {
		columns = append(columns, fi.Name)
	}
	if err := logq.Write(stdout, f, columns, rows); err != nil {
		fmt.Fprintf(stderr, "logq: %v\n", err)
		return 1
	}
	return 0
}

func sortUsage(w io.Writer) {
	fmt.Fprint(w, `usage: logq sort --by <field> [--desc] [files...]

Print all records ordered by the top-level <field>. Two JSON-number values
compare numerically; any other pair compares by canonical JSON rendering,
bytewise. Records missing the field sort last, keeping input order among
themselves, regardless of --desc. The sort is stable: ties keep input order.
`)
}
