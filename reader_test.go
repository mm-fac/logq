package logq

import (
	"os"
	"strings"
	"testing"
)

func TestReadLenientSkipsAndCounts(t *testing.T) {
	f, err := os.Open("testdata/mixed.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	res, err := Read(f, false)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	// mixed.jsonl: 4 valid objects, 2 malformed (non-JSON + array), 1 blank.
	if len(res.Records) != 4 {
		t.Errorf("records = %d, want 4", len(res.Records))
	}
	if res.Skipped != 2 {
		t.Errorf("skipped = %d, want 2", res.Skipped)
	}
}

func TestReadStrictErrorsOnMalformed(t *testing.T) {
	f, err := os.Open("testdata/mixed.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	res, err := Read(f, true)
	if err == nil {
		t.Fatalf("Read strict = %v, want error", res)
	}
	// The first malformed line is line 2; the error should identify it.
	if !strings.Contains(err.Error(), "line 2") {
		t.Errorf("error = %q, want it to mention line 2", err)
	}
}

func TestReadStrictCleanInputSucceeds(t *testing.T) {
	in := strings.NewReader("{\"a\":1}\n\n{\"b\":2}\n")
	res, err := Read(in, true)
	if err != nil {
		t.Fatalf("Read strict clean: %v", err)
	}
	if len(res.Records) != 2 || res.Skipped != 0 {
		t.Errorf("records=%d skipped=%d, want 2 and 0", len(res.Records), res.Skipped)
	}
}

func TestReadEmptyInput(t *testing.T) {
	res, err := Read(strings.NewReader(""), false)
	if err != nil {
		t.Fatalf("Read empty: %v", err)
	}
	if len(res.Records) != 0 || res.Skipped != 0 {
		t.Errorf("records=%d skipped=%d, want 0 and 0", len(res.Records), res.Skipped)
	}
}
