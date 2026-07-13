package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/mm-fac/logq"
)

func runStats(args []string, format string, strict bool, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("stats", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&format, "format", format, "output format: table|json|logfmt")
	fs.BoolVar(&strict, "strict", strict, "treat malformed input lines as a fatal error")
	groupBy := fs.String("group-by", "", "field to group records by")
	field := fs.String("field", "", "numeric field to aggregate")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *groupBy == "" {
		fmt.Fprintln(stderr, "logq: stats requires --group-by")
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

	columns := []string{*groupBy, "count"}
	if *field != "" {
		columns = append(columns, "min", "max", "sum", "avg", "skipped")
	}
	var rows []*logq.Record
	for _, sg := range logq.Stats(res.Records, *groupBy, *field) {
		row := logq.NewRecord().
			Set(*groupBy, sg.Value).
			Set("count", sg.Count)
		if *field != "" {
			row.Set("min", sg.Min).
				Set("max", sg.Max).
				Set("sum", sg.Sum).
				Set("avg", sg.Avg).
				Set("skipped", sg.Skipped)
		}
		rows = append(rows, row)
	}
	if err := logq.Write(stdout, f, columns, rows); err != nil {
		fmt.Fprintf(stderr, "logq: %v\n", err)
		return 1
	}
	return 0
}
