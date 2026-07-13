package main

import (
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
