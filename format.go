package logq

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Format is an output rendering for a set of rows. Every subcommand renders its
// results through Write so that --format is honored uniformly.
type Format int

const (
	// FormatTable is aligned, human-readable columns (the default).
	FormatTable Format = iota
	// FormatJSON is one compact JSON object per row (JSON-lines), symmetric
	// with the tool's input so output can be piped back in.
	FormatJSON
	// FormatLogfmt is `key=value` pairs per row.
	FormatLogfmt
)

// ParseFormat maps a --format string to a Format.
func ParseFormat(s string) (Format, error) {
	switch s {
	case "table":
		return FormatTable, nil
	case "json":
		return FormatJSON, nil
	case "logfmt":
		return FormatLogfmt, nil
	default:
		return 0, fmt.Errorf("unknown format %q (want table, json, or logfmt)", s)
	}
}

// String returns the canonical flag spelling of the format.
func (f Format) String() string {
	switch f {
	case FormatTable:
		return "table"
	case FormatJSON:
		return "json"
	case FormatLogfmt:
		return "logfmt"
	default:
		return "unknown"
	}
}

// Write renders rows to w in the given format. columns is the ordered set of
// column names (their union across rows); it drives table headers, logfmt key
// order, and JSON key order. A row missing a column is rendered as an empty
// cell (table) or an omitted key (json/logfmt), which distinguishes an absent
// field from one whose value is null.
func Write(w io.Writer, f Format, columns []string, rows []*Record) error {
	switch f {
	case FormatTable:
		return writeTable(w, columns, rows)
	case FormatJSON:
		return writeJSON(w, columns, rows)
	case FormatLogfmt:
		return writeLogfmt(w, columns, rows)
	default:
		return fmt.Errorf("unsupported format %d", int(f))
	}
}

func writeTable(w io.Writer, columns []string, rows []*Record) error {
	widths := make([]int, len(columns))
	for i, c := range columns {
		widths[i] = len(c)
	}
	cells := make([][]string, len(rows))
	for r, rec := range rows {
		cells[r] = make([]string, len(columns))
		for i, c := range columns {
			s := ""
			if v, ok := rec.Get(c); ok {
				s = cellString(v)
			}
			cells[r][i] = s
			if len(s) > widths[i] {
				widths[i] = len(s)
			}
		}
	}

	var b strings.Builder
	writeRow := func(vals []string) {
		for i, v := range vals {
			b.WriteString(v)
			if i < len(vals)-1 {
				b.WriteString(strings.Repeat(" ", widths[i]-len(v)+2))
			}
		}
		b.WriteByte('\n')
	}
	if len(columns) > 0 {
		writeRow(columns)
	}
	for _, row := range cells {
		writeRow(row)
	}
	_, err := io.WriteString(w, b.String())
	return err
}

func writeLogfmt(w io.Writer, columns []string, rows []*Record) error {
	var b strings.Builder
	for _, rec := range rows {
		first := true
		for _, c := range columns {
			v, ok := rec.Get(c)
			if !ok {
				continue
			}
			if !first {
				b.WriteByte(' ')
			}
			first = false
			b.WriteString(c)
			b.WriteByte('=')
			b.WriteString(logfmtValue(v))
		}
		b.WriteByte('\n')
	}
	_, err := io.WriteString(w, b.String())
	return err
}

func writeJSON(w io.Writer, columns []string, rows []*Record) error {
	var b strings.Builder
	for _, rec := range rows {
		b.WriteByte('{')
		first := true
		for _, c := range columns {
			v, ok := rec.Get(c)
			if !ok {
				continue
			}
			if !first {
				b.WriteByte(',')
			}
			first = false
			key, err := json.Marshal(c)
			if err != nil {
				return err
			}
			b.Write(key)
			b.WriteByte(':')
			val, err := json.Marshal(v)
			if err != nil {
				return err
			}
			b.Write(val)
		}
		b.WriteString("}\n")
	}
	_, err := io.WriteString(w, b.String())
	return err
}

// cellString renders a value for table output (and is the basis for logfmt).
func cellString(v any) string {
	switch t := v.(type) {
	case nil:
		return "null"
	case bool:
		return strconv.FormatBool(t)
	case json.Number:
		return t.String()
	case float64:
		return strconv.FormatFloat(t, 'g', -1, 64)
	case string:
		return t
	case []string:
		return strings.Join(t, ",")
	default:
		// Nested objects/arrays and anything else: compact JSON.
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(b)
	}
}

// logfmtValue renders a value as a logfmt token, quoting when the rendered form
// is empty or contains characters that would break `key=value` parsing.
func logfmtValue(v any) string {
	s := cellString(v)
	if s == "" || strings.ContainsAny(s, " \t\"=") {
		return strconv.Quote(s)
	}
	return s
}
