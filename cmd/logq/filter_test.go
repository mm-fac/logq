package main

import (
	"strings"
	"testing"
)

// filterSample exercises numeric, string, bool, and missing-field paths.
const filterSample = `{"level":"info","code":200,"msg":"start"}
{"level":"error","code":500,"msg":"boom","retry":true}
{"level":"info","code":200,"msg":"ok","latency":12.5}
{"level":"debug","msg":"trace"}
`

func TestFilterStringEquality(t *testing.T) {
	code, out, errOut := runCLI(t, []string{"--format", "json", "filter", "level==info"}, filterSample)
	if code != 0 {
		t.Fatalf("exit = %d, stderr=%q", code, errOut)
	}
	lines := nonEmptyLines(out)
	if len(lines) != 2 {
		t.Fatalf("got %d rows, want 2:\n%s", len(lines), out)
	}
	for _, l := range lines {
		if !strings.Contains(l, `"level":"info"`) {
			t.Errorf("row does not match level==info: %q", l)
		}
	}
}

func TestFilterNumericComparison(t *testing.T) {
	code, out, errOut := runCLI(t, []string{"--format", "json", "filter", "code>=300"}, filterSample)
	if code != 0 {
		t.Fatalf("exit = %d, stderr=%q", code, errOut)
	}
	lines := nonEmptyLines(out)
	if len(lines) != 1 {
		t.Fatalf("got %d rows, want 1:\n%s", len(lines), out)
	}
	if !strings.Contains(lines[0], `"code":500`) {
		t.Errorf("row = %q, want code 500", lines[0])
	}
}

func TestFilterSubstring(t *testing.T) {
	code, out, _ := runCLI(t, []string{"--format", "json", "filter", "msg~oo"}, filterSample)
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	lines := nonEmptyLines(out)
	// "boom" contains "oo"; nothing else does.
	if len(lines) != 1 || !strings.Contains(lines[0], `"msg":"boom"`) {
		t.Fatalf("got %v, want single boom row", lines)
	}
}

func TestFilterMultiplePredicatesAnd(t *testing.T) {
	code, out, _ := runCLI(t, []string{"--format", "json", "filter", "level==info", "code==200"}, filterSample)
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if got := len(nonEmptyLines(out)); got != 2 {
		t.Fatalf("got %d rows, want 2:\n%s", got, out)
	}
}

func TestFilterMissingFieldExcludes(t *testing.T) {
	// The debug record has no "code" key, so it cannot match a code predicate.
	code, out, _ := runCLI(t, []string{"--format", "json", "filter", "code==200"}, filterSample)
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if strings.Contains(out, `"level":"debug"`) {
		t.Errorf("debug record (missing code) should be excluded:\n%s", out)
	}
	if got := len(nonEmptyLines(out)); got != 2 {
		t.Fatalf("got %d rows, want 2:\n%s", got, out)
	}
}

func TestFilterBadPredicateIsUsageError(t *testing.T) {
	for _, arg := range []string{"==info", "code>", "level=info"} {
		code, _, errOut := runCLI(t, []string{"filter", arg}, filterSample)
		if arg == "level=info" {
			// A single '=' has no operator, so it is taken as a file path and
			// fails to open (runtime error), still non-zero.
			if code == 0 {
				t.Errorf("filter %q exit = 0, want non-zero", arg)
			}
			continue
		}
		if code != 2 {
			t.Errorf("filter %q exit = %d, want 2", arg, code)
		}
		if !strings.Contains(errOut, "invalid predicate") || !strings.Contains(errOut, "usage:") {
			t.Errorf("filter %q stderr = %q, want invalid-predicate usage message", arg, errOut)
		}
	}
}

func TestFilterNoPredicateIsUsageError(t *testing.T) {
	code, _, errOut := runCLI(t, []string{"filter"}, filterSample)
	if code != 2 {
		t.Errorf("exit = %d, want 2", code)
	}
	if !strings.Contains(errOut, "requires at least one predicate") {
		t.Errorf("stderr = %q", errOut)
	}
}

func TestFilterHonorsStrict(t *testing.T) {
	in := "{\"a\":1}\nnot json\n{\"a\":2}\n"
	code, _, errOut := runCLI(t, []string{"--strict", "filter", "a==1"}, in)
	if code == 0 {
		t.Fatalf("exit = 0, want non-zero")
	}
	if !strings.Contains(errOut, "line 2") {
		t.Errorf("stderr = %q, want line 2", errOut)
	}
}

func TestFilterSkipsMalformedAndReports(t *testing.T) {
	// mixed.jsonl has 2 malformed lines among valid records.
	code, out, errOut := runCLI(t, []string{"--format", "json", "filter", "../../testdata/mixed.jsonl", "level==info"}, "")
	if code != 0 {
		t.Fatalf("exit = %d, stderr=%q", code, errOut)
	}
	if !strings.Contains(errOut, "skipped 2 malformed line") {
		t.Errorf("stderr = %q, want skipped 2", errOut)
	}
	// Two info records in mixed.jsonl.
	if got := len(nonEmptyLines(out)); got != 2 {
		t.Fatalf("got %d rows, want 2:\n%s", got, out)
	}
}

func TestFilterTableFormat(t *testing.T) {
	code, out, errOut := runCLI(t, []string{"filter", "level==error"}, filterSample)
	if code != 0 {
		t.Fatalf("exit = %d, stderr=%q", code, errOut)
	}
	// Header row plus one matched record; columns are the union of the matched
	// record's keys in first-seen order.
	lines := nonEmptyLines(out)
	if len(lines) != 2 {
		t.Fatalf("got %d table lines, want header + 1 row:\n%s", len(lines), out)
	}
	header := strings.Fields(lines[0])
	wantCols := []string{"level", "code", "msg", "retry"}
	if strings.Join(header, ",") != strings.Join(wantCols, ",") {
		t.Errorf("header columns = %v, want %v", header, wantCols)
	}
	if !strings.Contains(lines[1], "error") || !strings.Contains(lines[1], "boom") {
		t.Errorf("table row missing matched values: %q", lines[1])
	}
}

func nonEmptyLines(s string) []string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if strings.TrimSpace(l) != "" {
			out = append(out, l)
		}
	}
	return out
}
