package main

import (
	"strings"
	"testing"
)

// nestedSample exercises dotted-path access end-to-end through the CLI: records
// nest a user object and a req object, and the last record's user is a scalar so
// traversal through a non-object resolves to "missing".
const nestedSample = `{"user":{"role":"admin"},"req":{"ms":10}}
{"user":{"role":"guest"},"req":{"ms":20}}
{"user":{"role":"admin"},"req":{"ms":30}}
{"user":"scalar","req":{"ms":40}}
`

// TestNestedFieldsAcrossSubcommands drives each field-taking subcommand with a
// dotted path and asserts the rendered output, proving all four adopt the shared
// resolver end-to-end.
func TestNestedFieldsAcrossSubcommands(t *testing.T) {
	t.Run("filter", func(t *testing.T) {
		code, out, errOut := runCLI(t, []string{"--format", "logfmt", "filter", "user.role==admin"}, nestedSample)
		if code != 0 {
			t.Fatalf("exit = %d, stderr=%q", code, errOut)
		}
		// Two admin records match; the scalar-user record does not.
		if got := strings.Count(out, "\n"); got != 2 {
			t.Errorf("matched %d lines, want 2:\n%s", got, out)
		}
		if strings.Contains(out, "guest") || strings.Contains(out, "scalar") {
			t.Errorf("unexpected non-admin record in output:\n%s", out)
		}
	})

	t.Run("stats", func(t *testing.T) {
		code, out, errOut := runCLI(t, []string{"stats", "--group-by", "user.role", "--field", "req.ms"}, nestedSample)
		if code != 0 {
			t.Fatalf("exit = %d, stderr=%q", code, errOut)
		}
		// The scalar-user record resolves user.role to "missing" (traversal
		// through a non-object), so it groups under null just like any absent
		// group-by field; null sorts first by canonical key.
		want := "user.role  count  min  max  sum  avg  skipped\n" +
			"null       1      40   40   40   40   0\n" +
			"admin      2      10   30   40   20   0\n" +
			"guest      1      20   20   20   20   0\n"
		if out != want {
			t.Errorf("stats output =\n%q\nwant\n%q", out, want)
		}
	})

	t.Run("distinct", func(t *testing.T) {
		code, out, errOut := runCLI(t, []string{"distinct", "user.role"}, nestedSample)
		if code != 0 {
			t.Fatalf("exit = %d, stderr=%q", code, errOut)
		}
		want := "value  count\n" +
			"admin  2\n" +
			"guest  1\n"
		if out != want {
			t.Errorf("distinct output =\n%q\nwant\n%q", out, want)
		}
		// The scalar-user record has no user.role, so it is reported as skipped.
		if !strings.Contains(errOut, "skipped 1") {
			t.Errorf("stderr = %q, want a skipped-1 report", errOut)
		}
	})

	t.Run("sort", func(t *testing.T) {
		code, out, errOut := runCLI(t, []string{"--format", "logfmt", "sort", "--by", "req.ms", "--desc"}, nestedSample)
		if code != 0 {
			t.Fatalf("exit = %d, stderr=%q", code, errOut)
		}
		// Descending by the nested numeric req.ms: 40, 30, 20, 10.
		want := `user=scalar req="{\"ms\":40}"` + "\n" +
			`user="{\"role\":\"admin\"}" req="{\"ms\":30}"` + "\n" +
			`user="{\"role\":\"guest\"}" req="{\"ms\":20}"` + "\n" +
			`user="{\"role\":\"admin\"}" req="{\"ms\":10}"` + "\n"
		if out != want {
			t.Errorf("sort output =\n%q\nwant\n%q", out, want)
		}
	})
}
