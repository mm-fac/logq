package main

import (
	"fmt"
	"strings"
	"testing"
)

func runCLI(t *testing.T, args []string, stdin string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errb strings.Builder
	code = run(args, strings.NewReader(stdin), &out, &errb)
	return code, out.String(), errb.String()
}

const sample = `{"level":"info","code":200,"msg":"start"}
{"level":"error","code":"oops","msg":"boom","retry":true}
{"level":"info","latency":12.5}
`

func TestFieldsTableDefault(t *testing.T) {
	code, out, errOut := runCLI(t, []string{"fields"}, sample)
	if code != 0 {
		t.Fatalf("exit = %d, stderr=%q", code, errOut)
	}
	want := "field    types          count\n" +
		"level    string         3\n" +
		"code     number,string  2\n" +
		"msg      string         2\n" +
		"retry    bool           1\n" +
		"latency  number         1\n"
	if out != want {
		t.Errorf("stdout =\n%q\nwant\n%q", out, want)
	}
	if errOut != "" {
		t.Errorf("stderr = %q, want empty", errOut)
	}
}

func TestFieldsJSONAndLogfmt(t *testing.T) {
	code, out, _ := runCLI(t, []string{"--format", "json", "fields"}, sample)
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.HasPrefix(out, `{"field":"level","types":["string"],"count":3}`) {
		t.Errorf("json stdout = %q", out)
	}

	// Flag also accepted after the subcommand.
	code, out, _ = runCLI(t, []string{"fields", "--format", "logfmt"}, sample)
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.HasPrefix(out, "field=level types=string count=3\n") {
		t.Errorf("logfmt stdout = %q", out)
	}
}

func TestFieldsLogfmtEscapesFieldNameNewline(t *testing.T) {
	code, out, errOut := runCLI(t, []string{"--format", "logfmt", "fields"}, "{\"line\\nbreak\":1}\n")
	if code != 0 {
		t.Fatalf("exit = %d, stderr=%q", code, errOut)
	}
	want := "field=\"line\\nbreak\" types=number count=1\n"
	if out != want {
		t.Errorf("stdout = %q, want %q", out, want)
	}
	if strings.Count(out, "\n") != 1 {
		t.Errorf("stdout contains a record-breaking newline: %q", out)
	}
}

func TestSkippedSummaryOnStderr(t *testing.T) {
	in := "{\"a\":1}\nnot json\n{\"a\":2}\n"
	code, out, errOut := runCLI(t, []string{"fields"}, in)
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(errOut, "skipped 1 malformed line") {
		t.Errorf("stderr = %q, want skipped summary", errOut)
	}
	if !strings.Contains(out, "field") { // still produced output
		t.Errorf("stdout = %q", out)
	}
}

func TestStrictExitsNonZero(t *testing.T) {
	in := "{\"a\":1}\nnot json\n"
	code, out, errOut := runCLI(t, []string{"--strict", "fields"}, in)
	if code == 0 {
		t.Fatalf("exit = 0, want non-zero; stdout=%q", out)
	}
	if !strings.Contains(errOut, "line 2") {
		t.Errorf("stderr = %q, want line 2", errOut)
	}
}

func TestFilesConcatenatedInOrder(t *testing.T) {
	// a.jsonl has a record with no trailing newline; b.jsonl follows. The
	// inserted separator must keep them as distinct records.
	code, out, errOut := runCLI(t, []string{"fields", "../../testdata/a.jsonl", "../../testdata/b.jsonl"}, "")
	if code != 0 {
		t.Fatalf("exit = %d stderr=%q", code, errOut)
	}
	// n appears in all 3 records, src in all 3.
	if !strings.Contains(out, "n") || !strings.Contains(out, "src") {
		t.Errorf("stdout = %q", out)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	// header + 2 field rows (n, src)
	if len(lines) != 3 {
		t.Fatalf("got %d lines, want 3:\n%s", len(lines), out)
	}
	for _, l := range lines[1:] { // each field appears in all 3 records
		if !strings.HasSuffix(l, "3") {
			t.Errorf("expected count 3 in row %q", l)
		}
	}
}

// numbered builds a JSON-lines input of count records {"id":1}..{"id":count}.
func numbered(count int) string {
	var b strings.Builder
	for i := 1; i <= count; i++ {
		fmt.Fprintf(&b, "{\"id\":%d}\n", i)
	}
	return b.String()
}

func TestTailDefaultKeepsLast10(t *testing.T) {
	// 12 records; the default -n 10 keeps the last 10 (id 3..12), in order.
	code, out, errOut := runCLI(t, []string{"--format", "logfmt", "tail"}, numbered(12))
	if code != 0 {
		t.Fatalf("exit = %d, stderr=%q", code, errOut)
	}
	var want strings.Builder
	for i := 3; i <= 12; i++ {
		fmt.Fprintf(&want, "id=%d\n", i)
	}
	if out != want.String() {
		t.Errorf("stdout =\n%q\nwant\n%q", out, want.String())
	}
	if errOut != "" {
		t.Errorf("stderr = %q, want empty", errOut)
	}
}

func TestTailNGreaterThanCountPrintsAll(t *testing.T) {
	// Only 3 records, ask for 100: every record is printed, none duplicated.
	code, out, errOut := runCLI(t, []string{"--format", "logfmt", "tail", "-n", "100"}, numbered(3))
	if code != 0 {
		t.Fatalf("exit = %d, stderr=%q", code, errOut)
	}
	want := "id=1\nid=2\nid=3\n"
	if out != want {
		t.Errorf("stdout = %q, want %q", out, want)
	}
}

func TestTailHonorsFormatAndStrict(t *testing.T) {
	// Malformed line without --strict is skipped and summarized on stderr; the
	// last record still renders through the shared JSON formatter.
	in := "{\"a\":1}\nnot json\n{\"a\":2}\n"
	code, out, errOut := runCLI(t, []string{"--format", "json", "tail", "-n", "1"}, in)
	if code != 0 {
		t.Fatalf("exit = %d, stderr=%q", code, errOut)
	}
	if out != "{\"a\":2}\n" {
		t.Errorf("stdout = %q, want last record as json", out)
	}
	if !strings.Contains(errOut, "skipped 1 malformed line") {
		t.Errorf("stderr = %q, want skipped summary", errOut)
	}
}

func TestTailNonPositiveNIsUsageError(t *testing.T) {
	for _, arg := range []string{"0", "-3"} {
		code, out, errOut := runCLI(t, []string{"tail", "-n", arg}, numbered(3))
		if code == 0 {
			t.Errorf("-n %s: exit = 0, want non-zero; stdout=%q", arg, out)
		}
		if out != "" {
			t.Errorf("-n %s: stdout = %q, want no output", arg, out)
		}
		if !strings.Contains(errOut, "positive integer") {
			t.Errorf("-n %s: stderr = %q, want usage message", arg, errOut)
		}
	}
}

func TestHeadDefaultKeepsFirst10(t *testing.T) {
	// 12 records; the default -n 10 keeps the first 10 (id 1..10), in order.
	code, out, errOut := runCLI(t, []string{"--format", "logfmt", "head"}, numbered(12))
	if code != 0 {
		t.Fatalf("exit = %d, stderr=%q", code, errOut)
	}
	var want strings.Builder
	for i := 1; i <= 10; i++ {
		fmt.Fprintf(&want, "id=%d\n", i)
	}
	if out != want.String() {
		t.Errorf("stdout =\n%q\nwant\n%q", out, want.String())
	}
	if errOut != "" {
		t.Errorf("stderr = %q, want empty", errOut)
	}
}

func TestHeadNHonored(t *testing.T) {
	// -n 2 keeps only the first two records.
	code, out, errOut := runCLI(t, []string{"--format", "logfmt", "head", "-n", "2"}, numbered(5))
	if code != 0 {
		t.Fatalf("exit = %d, stderr=%q", code, errOut)
	}
	if out != "id=1\nid=2\n" {
		t.Errorf("stdout = %q, want first two records", out)
	}
}

func TestHeadNGreaterThanCountPrintsAll(t *testing.T) {
	// Only 3 records, ask for 100: every record is printed, none duplicated.
	code, out, errOut := runCLI(t, []string{"--format", "logfmt", "head", "-n", "100"}, numbered(3))
	if code != 0 {
		t.Fatalf("exit = %d, stderr=%q", code, errOut)
	}
	want := "id=1\nid=2\nid=3\n"
	if out != want {
		t.Errorf("stdout = %q, want %q", out, want)
	}
}

func TestHeadHonorsFormats(t *testing.T) {
	// The first record renders through each of the three shared formatters.
	in := numbered(3)
	cases := []struct{ format, want string }{
		{"table", "id\n1\n"},
		{"json", "{\"id\":1}\n"},
		{"logfmt", "id=1\n"},
	}
	for _, tc := range cases {
		code, out, errOut := runCLI(t, []string{"--format", tc.format, "head", "-n", "1"}, in)
		if code != 0 {
			t.Fatalf("%s: exit = %d, stderr=%q", tc.format, code, errOut)
		}
		if out != tc.want {
			t.Errorf("%s: stdout = %q, want %q", tc.format, out, tc.want)
		}
	}
}

func TestHeadHonorsStrictAndSkip(t *testing.T) {
	// Malformed line without --strict is skipped and summarized on stderr; the
	// first valid record still renders. With --strict it becomes a fatal error.
	in := "not json\n{\"a\":1}\n{\"a\":2}\n"
	code, out, errOut := runCLI(t, []string{"--format", "json", "head", "-n", "1"}, in)
	if code != 0 {
		t.Fatalf("exit = %d, stderr=%q", code, errOut)
	}
	if out != "{\"a\":1}\n" {
		t.Errorf("stdout = %q, want first valid record as json", out)
	}
	if !strings.Contains(errOut, "skipped 1 malformed line") {
		t.Errorf("stderr = %q, want skipped summary", errOut)
	}

	code, out, errOut = runCLI(t, []string{"--strict", "head"}, in)
	if code == 0 {
		t.Fatalf("strict: exit = 0, want non-zero; stdout=%q", out)
	}
	if !strings.Contains(errOut, "line 1") {
		t.Errorf("strict: stderr = %q, want line 1", errOut)
	}
}

func TestHeadNonPositiveNIsUsageError(t *testing.T) {
	for _, arg := range []string{"0", "-3"} {
		code, out, errOut := runCLI(t, []string{"head", "-n", arg}, numbered(3))
		if code == 0 {
			t.Errorf("-n %s: exit = 0, want non-zero; stdout=%q", arg, out)
		}
		if out != "" {
			t.Errorf("-n %s: stdout = %q, want no output", arg, out)
		}
		if !strings.Contains(errOut, "positive integer") {
			t.Errorf("-n %s: stderr = %q, want usage message", arg, errOut)
		}
	}
}

func TestHeadFromFileArg(t *testing.T) {
	// Input from a file argument rather than stdin.
	code, out, errOut := runCLI(t, []string{"--format", "logfmt", "head", "-n", "1", "../../testdata/events.jsonl"}, "")
	if code != 0 {
		t.Fatalf("exit = %d, stderr=%q", code, errOut)
	}
	if !strings.HasPrefix(out, "ts=2026-07-13T00:00:01Z") {
		t.Errorf("stdout = %q, want first event record", out)
	}
}

func TestBadFormatIsUsageError(t *testing.T) {
	code, _, errOut := runCLI(t, []string{"--format", "yaml", "fields"}, sample)
	if code != 2 {
		t.Errorf("exit = %d, want 2", code)
	}
	if !strings.Contains(errOut, "unknown format") {
		t.Errorf("stderr = %q", errOut)
	}
}

func TestUnknownCommand(t *testing.T) {
	code, _, errOut := runCLI(t, []string{"bogus"}, "")
	if code != 2 {
		t.Errorf("exit = %d, want 2", code)
	}
	if !strings.Contains(errOut, "unknown command") {
		t.Errorf("stderr = %q", errOut)
	}
}

func TestNoCommandShowsUsage(t *testing.T) {
	code, _, errOut := runCLI(t, nil, "")
	if code != 2 {
		t.Errorf("exit = %d, want 2", code)
	}
	if !strings.Contains(errOut, "usage:") {
		t.Errorf("stderr = %q", errOut)
	}
}

func TestMissingFileErrors(t *testing.T) {
	code, _, errOut := runCLI(t, []string{"fields", "does-not-exist.jsonl"}, "")
	if code != 1 {
		t.Errorf("exit = %d, want 1", code)
	}
	if errOut == "" {
		t.Errorf("stderr empty, want an error")
	}
}
