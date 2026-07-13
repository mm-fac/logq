package main

import (
	"strings"
	"testing"
)

// Numeric values compare numerically (not bytewise: "2" would sort after "10").
// Exercised across all three output formats.
func TestSortNumericAllFormats(t *testing.T) {
	in := "{\"n\":10}\n{\"n\":2}\n{\"n\":100}\n"

	code, out, errOut := runCLI(t, []string{"--format", "logfmt", "sort", "--by", "n"}, in)
	if code != 0 {
		t.Fatalf("logfmt exit = %d, stderr=%q", code, errOut)
	}
	if want := "n=2\nn=10\nn=100\n"; out != want {
		t.Errorf("logfmt stdout = %q, want %q", out, want)
	}

	code, out, errOut = runCLI(t, []string{"--format", "json", "sort", "--by", "n"}, in)
	if code != 0 {
		t.Fatalf("json exit = %d, stderr=%q", code, errOut)
	}
	if want := "{\"n\":2}\n{\"n\":10}\n{\"n\":100}\n"; out != want {
		t.Errorf("json stdout = %q, want %q", out, want)
	}

	// table (default): a single column is padded to its widest value.
	code, out, errOut = runCLI(t, []string{"sort", "--by", "n"}, in)
	if code != 0 {
		t.Fatalf("table exit = %d, stderr=%q", code, errOut)
	}
	if want := "n\n2\n10\n100\n"; out != want {
		t.Errorf("table stdout = %q, want %q", out, want)
	}
	if errOut != "" {
		t.Errorf("stderr = %q, want empty", errOut)
	}
}

// Numbers beyond float64 precision/range must order exactly end-to-end through
// run(): the large integers must not tie (finding #1) and the huge magnitudes
// must not fall back to bytewise order (finding #2).
func TestSortExactNumericThroughRun(t *testing.T) {
	in := "{\"n\":9007199254740993}\n{\"n\":9007199254740992}\n{\"n\":10e400}\n{\"n\":9e400}\n"
	code, out, errOut := runCLI(t, []string{"--format", "logfmt", "sort", "--by", "n"}, in)
	if code != 0 {
		t.Fatalf("exit = %d, stderr=%q", code, errOut)
	}
	want := "n=9007199254740992\nn=9007199254740993\nn=9e400\nn=10e400\n"
	if out != want {
		t.Errorf("stdout = %q, want %q", out, want)
	}
}

func TestSortStrings(t *testing.T) {
	in := "{\"s\":\"banana\"}\n{\"s\":\"apple\"}\n{\"s\":\"cherry\"}\n"
	code, out, errOut := runCLI(t, []string{"--format", "logfmt", "sort", "--by", "s"}, in)
	if code != 0 {
		t.Fatalf("exit = %d, stderr=%q", code, errOut)
	}
	if want := "s=apple\ns=banana\ns=cherry\n"; out != want {
		t.Errorf("stdout = %q, want %q", out, want)
	}
}

// A number against a non-number compares by canonical JSON rendering, bytewise:
// "1" (0x22) < 1 (0x31) < null (0x6e) < true (0x74). The JSON format shows the
// type distinction that the compare relies on.
func TestSortMixedTypesJSON(t *testing.T) {
	in := "{\"v\":true}\n{\"v\":1}\n{\"v\":\"1\"}\n{\"v\":null}\n"
	code, out, errOut := runCLI(t, []string{"--format", "json", "sort", "--by", "v"}, in)
	if code != 0 {
		t.Fatalf("exit = %d, stderr=%q", code, errOut)
	}
	want := "{\"v\":\"1\"}\n{\"v\":1}\n{\"v\":null}\n{\"v\":true}\n"
	if out != want {
		t.Errorf("stdout = %q, want %q", out, want)
	}
}

// Records missing the field sort last, keeping input order among themselves, and
// --desc reverses only the present records, not that rule.
func TestSortMissingLast(t *testing.T) {
	in := "{\"k\":3,\"tag\":\"a\"}\n{\"other\":\"x\"}\n{\"k\":1,\"tag\":\"b\"}\n{\"other\":\"y\"}\n"

	code, out, errOut := runCLI(t, []string{"--format", "logfmt", "sort", "--by", "k"}, in)
	if code != 0 {
		t.Fatalf("asc exit = %d, stderr=%q", code, errOut)
	}
	if want := "k=1 tag=b\nk=3 tag=a\nother=x\nother=y\n"; out != want {
		t.Errorf("asc stdout = %q, want %q", out, want)
	}

	code, out, errOut = runCLI(t, []string{"--format", "logfmt", "sort", "--by", "k", "--desc"}, in)
	if code != 0 {
		t.Fatalf("desc exit = %d, stderr=%q", code, errOut)
	}
	if want := "k=3 tag=a\nk=1 tag=b\nother=x\nother=y\n"; out != want {
		t.Errorf("desc stdout = %q, want %q", out, want)
	}
}

// Ties keep input order (stable), in both directions — the tag field tracks the
// original position of records sharing a sort value.
func TestSortStableTies(t *testing.T) {
	in := "{\"k\":1,\"tag\":\"a\"}\n{\"k\":1,\"tag\":\"b\"}\n{\"k\":1,\"tag\":\"c\"}\n"
	for _, args := range [][]string{
		{"--format", "logfmt", "sort", "--by", "k"},
		{"--format", "logfmt", "sort", "--by", "k", "--desc"},
	} {
		code, out, errOut := runCLI(t, args, in)
		if code != 0 {
			t.Fatalf("%v exit = %d, stderr=%q", args, code, errOut)
		}
		if want := "k=1 tag=a\nk=1 tag=b\nk=1 tag=c\n"; out != want {
			t.Errorf("%v stdout = %q, want %q", args, out, want)
		}
	}
}

func TestSortDescReversesComparison(t *testing.T) {
	in := "{\"n\":2}\n{\"n\":10}\n{\"n\":100}\n"
	code, out, errOut := runCLI(t, []string{"--format", "logfmt", "sort", "--by", "n", "--desc"}, in)
	if code != 0 {
		t.Fatalf("exit = %d, stderr=%q", code, errOut)
	}
	if want := "n=100\nn=10\nn=2\n"; out != want {
		t.Errorf("stdout = %q, want %q", out, want)
	}
}

func TestSortMissingByFlagIsUsageError(t *testing.T) {
	code, out, errOut := runCLI(t, []string{"sort"}, "{\"n\":1}\n")
	if code != 2 {
		t.Errorf("exit = %d, want 2", code)
	}
	if out != "" {
		t.Errorf("stdout = %q, want empty", out)
	}
	if !strings.Contains(errOut, "requires --by") {
		t.Errorf("stderr = %q, want --by requirement", errOut)
	}
	if !strings.Contains(errOut, "usage: logq sort") {
		t.Errorf("stderr = %q, want usage", errOut)
	}
}

// --strict is honored: a malformed line aborts with a non-zero exit; without it
// the line is skipped and summarized on stderr while sorting continues.
func TestSortHonorsStrict(t *testing.T) {
	in := "{\"n\":2}\nnot json\n{\"n\":1}\n"

	code, out, errOut := runCLI(t, []string{"--strict", "sort", "--by", "n"}, in)
	if code == 0 {
		t.Fatalf("strict exit = 0, want non-zero; stdout=%q", out)
	}
	if !strings.Contains(errOut, "line 2") {
		t.Errorf("strict stderr = %q, want line 2", errOut)
	}

	code, out, errOut = runCLI(t, []string{"--format", "logfmt", "sort", "--by", "n"}, in)
	if code != 0 {
		t.Fatalf("lenient exit = %d, stderr=%q", code, errOut)
	}
	if want := "n=1\nn=2\n"; out != want {
		t.Errorf("lenient stdout = %q, want %q", out, want)
	}
	if !strings.Contains(errOut, "skipped 1 malformed line") {
		t.Errorf("lenient stderr = %q, want skipped summary", errOut)
	}
}
