package logq

import (
	"encoding/json"
	"strings"
	"testing"
)

// sampleRows builds a small heterogeneous rowset: rows have differing keys so
// the tests exercise absent-column handling, and values cover string, number,
// bool, null, array, and nested object.
func sampleRows() ([]string, []*Record) {
	columns := []string{"field", "types", "count"}
	rows := []*Record{
		NewRecord().Set("field", "level").Set("types", []string{"string"}).Set("count", 3),
		NewRecord().Set("field", "code").Set("types", []string{"number", "string"}).Set("count", 2),
	}
	return columns, rows
}

func TestParseFormat(t *testing.T) {
	for s, want := range map[string]Format{"table": FormatTable, "json": FormatJSON, "logfmt": FormatLogfmt} {
		got, err := ParseFormat(s)
		if err != nil || got != want {
			t.Errorf("ParseFormat(%q) = %v, %v", s, got, err)
		}
		if got.String() != s {
			t.Errorf("Format(%v).String() = %q, want %q", got, got.String(), s)
		}
	}
	if _, err := ParseFormat("yaml"); err == nil {
		t.Error("ParseFormat(yaml) = nil error, want error")
	}
}

func TestWriteTable(t *testing.T) {
	columns, rows := sampleRows()
	var b strings.Builder
	if err := Write(&b, FormatTable, columns, rows); err != nil {
		t.Fatal(err)
	}
	want := "field  types          count\n" +
		"level  string         3\n" +
		"code   number,string  2\n"
	if b.String() != want {
		t.Errorf("table =\n%q\nwant\n%q", b.String(), want)
	}
}

func TestWriteLogfmt(t *testing.T) {
	columns, rows := sampleRows()
	var b strings.Builder
	if err := Write(&b, FormatLogfmt, columns, rows); err != nil {
		t.Fatal(err)
	}
	// A comma does not break key=value parsing, so it is not quoted.
	want := "field=level types=string count=3\n" +
		"field=code types=number,string count=2\n"
	if b.String() != want {
		t.Errorf("logfmt =\n%q\nwant\n%q", b.String(), want)
	}
}

func TestWriteLogfmtQuotesControlWhitespace(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "newline", value: "line\nbreak", want: `field="line\nbreak"` + "\n"},
		{name: "carriage return", value: "line\rbreak", want: `field="line\rbreak"` + "\n"},
		{name: "vertical tab", value: "line\vbreak", want: `field="line\vbreak"` + "\n"},
		{name: "form feed", value: "line\fbreak", want: `field="line\fbreak"` + "\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var b strings.Builder
			row := NewRecord().Set("field", tt.value)
			if err := Write(&b, FormatLogfmt, []string{"field"}, []*Record{row}); err != nil {
				t.Fatal(err)
			}
			if got := b.String(); got != tt.want {
				t.Errorf("logfmt = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWriteJSON(t *testing.T) {
	columns, rows := sampleRows()
	var b strings.Builder
	if err := Write(&b, FormatJSON, columns, rows); err != nil {
		t.Fatal(err)
	}
	want := `{"field":"level","types":["string"],"count":3}` + "\n" +
		`{"field":"code","types":["number","string"],"count":2}` + "\n"
	if b.String() != want {
		t.Errorf("json =\n%q\nwant\n%q", b.String(), want)
	}
	// Each line must be valid JSON.
	for _, line := range strings.Split(strings.TrimSpace(b.String()), "\n") {
		var v any
		if err := json.Unmarshal([]byte(line), &v); err != nil {
			t.Errorf("invalid JSON line %q: %v", line, err)
		}
	}
}

// TestWriteValueTypesAndAbsence checks scalar rendering and that a missing
// column is an empty cell (table) / omitted key (logfmt, json).
func TestWriteValueTypesAndAbsence(t *testing.T) {
	columns := []string{"s", "n", "b", "z", "arr", "obj"}
	rows := []*Record{
		NewRecord().
			Set("s", "hi there"). // space -> logfmt quotes it
			Set("n", json.Number("12.5")).
			Set("b", true).
			Set("z", nil).
			Set("arr", []any{json.Number("1"), "x"}).
			Set("obj", map[string]any{"k": json.Number("1")}),
		NewRecord().Set("s", "only-s"), // all other columns absent
	}

	var tbl strings.Builder
	if err := Write(&tbl, FormatTable, columns, rows); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(tbl.String(), "null") {
		t.Errorf("table missing null rendering:\n%s", tbl.String())
	}

	var lf strings.Builder
	if err := Write(&lf, FormatLogfmt, columns, rows); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(lf.String()), "\n")
	// Nested values render as compact JSON; because they contain quotes they
	// are themselves quoted so the logfmt stays parseable.
	if lines[0] != `s="hi there" n=12.5 b=true z=null arr="[1,\"x\"]" obj="{\"k\":1}"` {
		t.Errorf("logfmt row0 = %q", lines[0])
	}
	if lines[1] != "s=only-s" { // absent keys omitted
		t.Errorf("logfmt row1 = %q, want only s=only-s", lines[1])
	}

	var js strings.Builder
	if err := Write(&js, FormatJSON, columns, rows); err != nil {
		t.Fatal(err)
	}
	jsLines := strings.Split(strings.TrimSpace(js.String()), "\n")
	if jsLines[1] != `{"s":"only-s"}` { // absent keys omitted, present-null retained above
		t.Errorf("json row1 = %q, want {\"s\":\"only-s\"}", jsLines[1])
	}
	if !strings.Contains(jsLines[0], `"z":null`) {
		t.Errorf("json row0 = %q, want it to retain present null", jsLines[0])
	}
}
