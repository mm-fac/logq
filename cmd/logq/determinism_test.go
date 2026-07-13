package main

import (
	"testing"
)

// TestDeterministicOutput is the CI determinism check (matrix row 7). It runs
// every subcommand in every format three times on the same committed fixture and
// asserts byte-identical stdout each time. Because it is an ordinary `go test`,
// the existing CI job (`go test ./...`) executes it on every push and PR without
// touching the protected .github/ workflow.
func TestDeterministicOutput(t *testing.T) {
	content := readFixture(t, eventsFixture)

	for _, sc := range subcommandCases() {
		for _, format := range outputFormats {
			t.Run(sc.name+"/"+format, func(t *testing.T) {
				args := append([]string{"--format", format}, sc.args...)

				code, first, errOut := runCLI(t, args, content)
				if code != 0 {
					t.Fatalf("exit = %d, stderr=%q", code, errOut)
				}
				const runs = 3
				for i := 1; i < runs; i++ {
					code, out, _ := runCLI(t, args, content)
					if code != 0 {
						t.Fatalf("run %d: exit = %d", i, code)
					}
					if out != first {
						t.Fatalf("run %d output differs from run 0:\nfirst=%q\ngot  =%q", i, first, out)
					}
				}

				// File input must also be stable across runs and match stdin.
				fileArgs := append(append([]string{}, args...), eventsFixture)
				for i := 0; i < runs; i++ {
					code, out, _ := runCLI(t, fileArgs, "")
					if code != 0 {
						t.Fatalf("file run %d: exit = %d", i, code)
					}
					if out != first {
						t.Fatalf("file run %d differs from stdin output:\nstdin=%q\nfile =%q", i, first, out)
					}
				}
			})
		}
	}
}
