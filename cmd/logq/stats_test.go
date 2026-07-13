package main

import (
	"strings"
	"testing"
)

const statsFixture = "../../testdata/stats.jsonl"

func TestStatsGroupByCountsTable(t *testing.T) {
	code, out, errOut := runCLI(t, []string{"stats", "--group-by", "level", statsFixture}, "")
	if code != 0 {
		t.Fatalf("exit = %d, stderr=%q", code, errOut)
	}
	want := "level  count\n" +
		"error  2\n" +
		"info   3\n"
	if out != want {
		t.Errorf("stdout =\n%q\nwant\n%q", out, want)
	}
	if errOut != "" {
		t.Errorf("stderr = %q, want empty", errOut)
	}
}

func TestStatsAggregatesAndSkipsNonNumericTable(t *testing.T) {
	code, out, errOut := runCLI(t, []string{"stats", "--group-by", "level", "--field", "latency", statsFixture}, "")
	if code != 0 {
		t.Fatalf("exit = %d, stderr=%q", code, errOut)
	}
	want := "level  count  min  max   sum  avg  skipped\n" +
		"error  2      30   30    30   30   1\n" +
		"info   3      7.5  12.5  20   10   1\n"
	if out != want {
		t.Errorf("stdout =\n%q\nwant\n%q", out, want)
	}
}

func TestStatsJSONAndLogfmt(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "json",
			args: []string{"--format", "json", "stats", "--group-by", "level", "--field", "latency", statsFixture},
			want: `{"level":"error","count":2,"min":30,"max":30,"sum":30,"avg":30,"skipped":1}` + "\n" +
				`{"level":"info","count":3,"min":7.5,"max":12.5,"sum":20,"avg":10,"skipped":1}` + "\n",
		},
		{
			name: "logfmt",
			args: []string{"stats", "--format", "logfmt", "--group-by", "level", statsFixture},
			want: "level=error count=2\n" +
				"level=info count=3\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, out, errOut := runCLI(t, tt.args, "")
			if code != 0 {
				t.Fatalf("exit = %d, stderr=%q", code, errOut)
			}
			if out != tt.want {
				t.Errorf("stdout =\n%q\nwant\n%q", out, tt.want)
			}
		})
	}
}

func TestStatsMalformedLineContract(t *testing.T) {
	in := "{\"level\":\"info\"}\nnot json\n{\"level\":\"info\"}\n"
	code, out, errOut := runCLI(t, []string{"stats", "--group-by", "level"}, in)
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(errOut, "skipped 1 malformed line") {
		t.Errorf("stderr = %q, want skipped summary", errOut)
	}
	if !strings.Contains(out, "info") {
		t.Errorf("stdout = %q, want output despite malformed line", out)
	}

	code, out, errOut = runCLI(t, []string{"--strict", "stats", "--group-by", "level"}, in)
	if code == 0 {
		t.Fatalf("exit = 0, want non-zero; stdout=%q", out)
	}
	if !strings.Contains(errOut, "line 2") {
		t.Errorf("stderr = %q, want line 2", errOut)
	}
}

func TestStatsRequiresGroupBy(t *testing.T) {
	code, _, errOut := runCLI(t, []string{"stats", statsFixture}, "")
	if code != 2 {
		t.Errorf("exit = %d, want 2", code)
	}
	if !strings.Contains(errOut, "requires --group-by") {
		t.Errorf("stderr = %q", errOut)
	}
}
