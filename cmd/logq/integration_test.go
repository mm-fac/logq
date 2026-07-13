package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// eventsFixture is the committed sample log documented in README.md. The
// integration tests drive the real run() entry point against it so the acceptance
// matrix is proven end-to-end rather than at the library boundary.
const eventsFixture = "../../testdata/events.jsonl"

// subcommandCase is one subcommand invoked with the minimal arguments that
// produce non-empty output against eventsFixture. args are the tokens that follow
// any global flags and precede the input source (stdin or a file path). col0 is
// the first output column/key, used to validate each rendered format.
type subcommandCase struct {
	name string
	args []string
	col0 string
}

func subcommandCases() []subcommandCase {
	return []subcommandCase{
		{name: "fields", args: []string{"fields"}, col0: "field"},
		{name: "filter", args: []string{"filter", "status>=200"}, col0: "ts"},
		{name: "stats", args: []string{"stats", "--group-by", "level", "--field", "ms"}, col0: "level"},
		{name: "tail", args: []string{"tail", "-n", "3"}, col0: "ts"},
	}
}

var outputFormats = []string{"table", "json", "logfmt"}

func readFixture(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	return string(b)
}

// TestAcceptanceMatrix drives every subcommand across every --format and both
// input sources (stdin and file args), asserting each combination succeeds,
// renders valid output for its format, and yields identical bytes whether the
// data arrives on stdin or as a file argument. This covers matrix rows 2 and 3.
func TestAcceptanceMatrix(t *testing.T) {
	content := readFixture(t, eventsFixture)

	for _, sc := range subcommandCases() {
		for _, format := range outputFormats {
			t.Run(sc.name+"/"+format, func(t *testing.T) {
				globalArgs := append([]string{"--format", format}, sc.args...)

				// Input via stdin (no file args).
				codeIn, outIn, errIn := runCLI(t, globalArgs, content)
				if codeIn != 0 {
					t.Fatalf("stdin: exit = %d, stderr=%q", codeIn, errIn)
				}
				if errIn != "" {
					t.Errorf("stdin: stderr = %q, want empty", errIn)
				}
				if strings.TrimSpace(outIn) == "" {
					t.Fatalf("stdin: empty stdout")
				}
				validateFormat(t, format, sc.col0, outIn)

				// Input via file argument.
				fileArgs := append(append([]string{}, globalArgs...), eventsFixture)
				codeFile, outFile, errFile := runCLI(t, fileArgs, "")
				if codeFile != 0 {
					t.Fatalf("file: exit = %d, stderr=%q", codeFile, errFile)
				}
				if errFile != "" {
					t.Errorf("file: stderr = %q, want empty", errFile)
				}

				// The same records from stdin or a file must render identically.
				if outIn != outFile {
					t.Errorf("stdin vs file output differ:\nstdin=%q\nfile =%q", outIn, outFile)
				}
			})
		}
	}
}

// TestDefaultFormatIsTable asserts that omitting --format is equivalent to
// --format table for every subcommand (matrix row 3, "default is table").
func TestDefaultFormatIsTable(t *testing.T) {
	content := readFixture(t, eventsFixture)
	for _, sc := range subcommandCases() {
		t.Run(sc.name, func(t *testing.T) {
			_, outDefault, _ := runCLI(t, sc.args, content)
			explicit := append([]string{"--format", "table"}, sc.args...)
			_, outTable, _ := runCLI(t, explicit, content)
			if outDefault != outTable {
				t.Errorf("default != table:\ndefault=%q\ntable  =%q", outDefault, outTable)
			}
		})
	}
}

// TestSharedMalformedContract asserts every subcommand honors the one malformed
// -line / --strict contract: lenient mode skips and counts malformed lines on
// stderr while still exiting 0 and producing output; --strict turns the first
// malformed line into a non-zero exit that names the line. This covers matrix
// row 4 across all subcommands from a single input.
func TestSharedMalformedContract(t *testing.T) {
	const in = "{\"level\":\"info\",\"ms\":5}\nnot json\n{\"level\":\"info\",\"ms\":7}\n"
	cases := []struct {
		name string
		args []string
	}{
		{"fields", []string{"fields"}},
		{"filter", []string{"filter", "level==info"}},
		{"stats", []string{"stats", "--group-by", "level"}},
		{"tail", []string{"tail"}},
	}
	for _, tc := range cases {
		t.Run(tc.name+"/lenient", func(t *testing.T) {
			code, out, errOut := runCLI(t, tc.args, in)
			if code != 0 {
				t.Fatalf("exit = %d, stderr=%q", code, errOut)
			}
			if !strings.Contains(errOut, "skipped 1 malformed line") {
				t.Errorf("stderr = %q, want skipped summary", errOut)
			}
			if strings.TrimSpace(out) == "" {
				t.Errorf("stdout empty, want output despite malformed line")
			}
		})
		t.Run(tc.name+"/strict", func(t *testing.T) {
			args := append([]string{"--strict"}, tc.args...)
			code, out, errOut := runCLI(t, args, in)
			if code == 0 {
				t.Fatalf("exit = 0, want non-zero; stdout=%q", out)
			}
			if out != "" {
				t.Errorf("stdout = %q, want empty on strict error", out)
			}
			if !strings.Contains(errOut, "line 2") {
				t.Errorf("stderr = %q, want line 2", errOut)
			}
		})
	}
}

// TestUsageErrorsExitNonZero covers matrix row 5: unknown subcommand, bad
// predicate, unknown flag, and bad --format each exit non-zero with a message on
// stderr and no output on stdout.
func TestUsageErrorsExitNonZero(t *testing.T) {
	content := readFixture(t, eventsFixture)
	cases := []struct {
		name       string
		args       []string
		stdin      string
		wantStderr string
	}{
		{"unknown-command", []string{"bogus"}, "", "unknown command"},
		{"bad-predicate", []string{"filter", "level=info"}, content, "invalid predicate"},
		{"unknown-flag", []string{"fields", "--nope"}, content, "not defined"},
		{"bad-format", []string{"--format", "yaml", "fields"}, content, "unknown format"},
		{"no-command", nil, "", "usage:"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code, out, errOut := runCLI(t, tc.args, tc.stdin)
			if code == 0 {
				t.Fatalf("exit = 0, want non-zero; stdout=%q", out)
			}
			if out != "" {
				t.Errorf("stdout = %q, want empty", out)
			}
			if !strings.Contains(errOut, tc.wantStderr) {
				t.Errorf("stderr = %q, want to contain %q", errOut, tc.wantStderr)
			}
		})
	}
}

// validateFormat checks that out is well-formed for the given format: JSON is
// one parseable object per line, logfmt is key=value lines, and table has a
// header row. col0 is the expected first column/key.
func validateFormat(t *testing.T, format, col0, out string) {
	t.Helper()
	lines := nonEmptyLines(out)
	if len(lines) == 0 {
		t.Fatalf("no output lines to validate")
	}
	switch format {
	case "json":
		for _, l := range lines {
			var obj map[string]json.RawMessage
			if err := json.Unmarshal([]byte(l), &obj); err != nil {
				t.Errorf("invalid JSON line %q: %v", l, err)
				continue
			}
			if _, ok := obj[col0]; !ok {
				t.Errorf("JSON line %q missing first key %q", l, col0)
			}
		}
	case "logfmt":
		for _, l := range lines {
			if !strings.Contains(l, "=") {
				t.Errorf("logfmt line %q has no key=value pair", l)
			}
			if strings.HasPrefix(l, " ") {
				t.Errorf("logfmt line %q has leading space", l)
			}
		}
		if !strings.HasPrefix(lines[0], col0+"=") {
			t.Errorf("logfmt first line %q does not start with %q", lines[0], col0+"=")
		}
	case "table":
		header := strings.Fields(lines[0])
		if len(header) == 0 || header[0] != col0 {
			t.Errorf("table header = %v, want first column %q", header, col0)
		}
	default:
		t.Fatalf("unknown format %q", format)
	}
}
