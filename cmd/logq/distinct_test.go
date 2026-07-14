package main

import (
	"strings"
	"testing"
)

const distinctSample = `{"level":"info"}
{"level":"warn"}
{"level":"info"}
{"level":"error"}
{"level":"info"}
`

func TestDistinctTableDefault(t *testing.T) {
	code, out, errOut := runCLI(t, []string{"distinct", "level"}, distinctSample)
	if code != 0 {
		t.Fatalf("exit = %d, stderr=%q", code, errOut)
	}
	want := "value  count\n" +
		"error  1\n" +
		"info   3\n" +
		"warn   1\n"
	if out != want {
		t.Errorf("stdout =\n%q\nwant\n%q", out, want)
	}
	if errOut != "" {
		t.Errorf("stderr = %q, want empty", errOut)
	}
}

func TestDistinctJSONAndLogfmt(t *testing.T) {
	code, out, errOut := runCLI(t, []string{"--format", "json", "distinct", "level"}, distinctSample)
	if code != 0 {
		t.Fatalf("exit = %d, stderr=%q", code, errOut)
	}
	wantJSON := `{"value":"error","count":1}
{"value":"info","count":3}
{"value":"warn","count":1}
`
	if out != wantJSON {
		t.Errorf("json stdout =\n%q\nwant\n%q", out, wantJSON)
	}

	// Flag also accepted after the subcommand (before the field argument).
	code, out, errOut = runCLI(t, []string{"distinct", "--format", "logfmt", "level"}, distinctSample)
	if code != 0 {
		t.Fatalf("exit = %d, stderr=%q", code, errOut)
	}
	wantLogfmt := "value=error count=1\nvalue=info count=3\nvalue=warn count=1\n"
	if out != wantLogfmt {
		t.Errorf("logfmt stdout =\n%q\nwant\n%q", out, wantLogfmt)
	}
}

// Values of different JSON types stay distinct even when they render alike; the
// JSON format shows the distinction (1 vs "1"), and rows sort by canonical JSON.
func TestDistinctMixedTypesJSON(t *testing.T) {
	in := "{\"v\":1}\n{\"v\":\"1\"}\n{\"v\":1}\n"
	code, out, errOut := runCLI(t, []string{"--format", "json", "distinct", "v"}, in)
	if code != 0 {
		t.Fatalf("exit = %d, stderr=%q", code, errOut)
	}
	want := `{"value":"1","count":1}
{"value":1,"count":2}
`
	if out != want {
		t.Errorf("stdout =\n%q\nwant\n%q", out, want)
	}
}

func TestDistinctMissingFieldSkipCount(t *testing.T) {
	in := "{\"a\":\"x\"}\n{\"b\":\"y\"}\n{\"a\":\"x\"}\n"
	code, out, errOut := runCLI(t, []string{"distinct", "a"}, in)
	if code != 0 {
		t.Fatalf("exit = %d, stderr=%q", code, errOut)
	}
	if !strings.Contains(errOut, `skipped 1 record(s) missing field "a"`) {
		t.Errorf("stderr = %q, want missing-field skip count", errOut)
	}
	// Output still produced for the records that had the field.
	if !strings.Contains(out, "x") {
		t.Errorf("stdout = %q, want the present value", out)
	}
}

func TestDistinctMissingFieldArgIsUsageError(t *testing.T) {
	code, out, errOut := runCLI(t, []string{"distinct"}, distinctSample)
	if code != 2 {
		t.Errorf("exit = %d, want 2", code)
	}
	if out != "" {
		t.Errorf("stdout = %q, want empty", out)
	}
	if !strings.Contains(errOut, "usage: logq distinct") {
		t.Errorf("stderr = %q, want usage", errOut)
	}
}

// --top N keeps the N most frequent values, ordered by count descending. The
// sample has info:3, warn:1, error:1; top-2 is info first, then the count-1
// tie broken by canonical JSON ascending ("error" < "warn") selects error.
func TestDistinctTopOrdersByCountDesc(t *testing.T) {
	code, out, errOut := runCLI(t, []string{"distinct", "level", "--top", "2"}, distinctSample)
	if code != 0 {
		t.Fatalf("exit = %d, stderr=%q", code, errOut)
	}
	want := "value  count\n" +
		"info   3\n" +
		"error  1\n"
	if out != want {
		t.Errorf("stdout =\n%q\nwant\n%q", out, want)
	}
	if errOut != "" {
		t.Errorf("stderr = %q, want empty", errOut)
	}
}

// When counts tie, rows break by the value's canonical JSON rendering ascending
// bytewise. Here a, b and c all occur twice, so top-3 lists them a, b, c.
func TestDistinctTopTieBreak(t *testing.T) {
	in := "{\"k\":\"c\"}\n{\"k\":\"a\"}\n{\"k\":\"b\"}\n{\"k\":\"a\"}\n{\"k\":\"b\"}\n{\"k\":\"c\"}\n"
	code, out, errOut := runCLI(t, []string{"distinct", "k", "--top", "3"}, in)
	if code != 0 {
		t.Fatalf("exit = %d, stderr=%q", code, errOut)
	}
	want := "value  count\n" +
		"a      2\n" +
		"b      2\n" +
		"c      2\n"
	if out != want {
		t.Errorf("stdout =\n%q\nwant\n%q", out, want)
	}
}

// N greater than the distinct-value count prints every value (min(N, count)),
// still in count-descending / canonical-ascending order.
func TestDistinctTopExceedsDistinctCount(t *testing.T) {
	code, out, errOut := runCLI(t, []string{"distinct", "level", "--top", "99"}, distinctSample)
	if code != 0 {
		t.Fatalf("exit = %d, stderr=%q", code, errOut)
	}
	want := "value  count\n" +
		"info   3\n" +
		"error  1\n" +
		"warn   1\n"
	if out != want {
		t.Errorf("stdout =\n%q\nwant\n%q", out, want)
	}
}

// --top honors all three output formats.
func TestDistinctTopFormats(t *testing.T) {
	code, out, errOut := runCLI(t, []string{"--format", "json", "distinct", "level", "--top", "2"}, distinctSample)
	if code != 0 {
		t.Fatalf("exit = %d, stderr=%q", code, errOut)
	}
	wantJSON := `{"value":"info","count":3}
{"value":"error","count":1}
`
	if out != wantJSON {
		t.Errorf("json stdout =\n%q\nwant\n%q", out, wantJSON)
	}

	code, out, errOut = runCLI(t, []string{"--format", "logfmt", "distinct", "level", "--top", "2"}, distinctSample)
	if code != 0 {
		t.Fatalf("exit = %d, stderr=%q", code, errOut)
	}
	wantLogfmt := "value=info count=3\nvalue=error count=1\n"
	if out != wantLogfmt {
		t.Errorf("logfmt stdout =\n%q\nwant\n%q", out, wantLogfmt)
	}
}

// --top composes with nested dot-path fields.
func TestDistinctTopNestedField(t *testing.T) {
	in := "{\"user\":{\"role\":\"admin\"}}\n" +
		"{\"user\":{\"role\":\"guest\"}}\n" +
		"{\"user\":{\"role\":\"admin\"}}\n"
	code, out, errOut := runCLI(t, []string{"distinct", "user.role", "--top", "1"}, in)
	if code != 0 {
		t.Fatalf("exit = %d, stderr=%q", code, errOut)
	}
	want := "value  count\n" +
		"admin  2\n"
	if out != want {
		t.Errorf("stdout =\n%q\nwant\n%q", out, want)
	}
}

func TestDistinctTopNonPositiveIsUsageError(t *testing.T) {
	for _, n := range []string{"0", "-1"} {
		code, out, errOut := runCLI(t, []string{"distinct", "level", "--top", n}, distinctSample)
		if code == 0 {
			t.Errorf("--top %s: exit = 0, want non-zero", n)
		}
		if out != "" {
			t.Errorf("--top %s: stdout = %q, want empty", n, out)
		}
		if !strings.Contains(errOut, "usage: logq distinct") {
			t.Errorf("--top %s: stderr = %q, want usage", n, errOut)
		}
	}
}

func TestDistinctUnknownFlagIsUsageError(t *testing.T) {
	code, out, errOut := runCLI(t, []string{"distinct", "--nope", "level"}, distinctSample)
	if code != 2 {
		t.Errorf("exit = %d, want 2", code)
	}
	if out != "" {
		t.Errorf("stdout = %q, want empty", out)
	}
	if !strings.Contains(errOut, "not defined") {
		t.Errorf("stderr = %q, want flag error", errOut)
	}
}
