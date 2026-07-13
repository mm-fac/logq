package logq

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
)

// maxLineBytes bounds a single input line. Log lines can be large, so the
// scanner buffer is generous; lines longer than this are reported as an error
// via Read rather than silently truncated.
const maxLineBytes = 16 * 1024 * 1024

// Result holds everything Read produced from an input stream.
type Result struct {
	// Records are the successfully parsed log entries, in input order.
	Records []*Record
	// Skipped counts malformed (non-parseable) lines dropped in lenient mode.
	// Blank/whitespace-only lines are ignored and never counted here.
	Skipped int
}

// Read consumes r as JSON-lines: one JSON object per line. Blank lines are
// ignored. In lenient mode (strict=false) malformed lines are skipped and
// tallied in Result.Skipped. In strict mode the first malformed line returns a
// non-nil error identifying the line number, and no Result is returned.
func Read(r io.Reader, strict bool) (*Result, error) {
	res := &Result{}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), maxLineBytes)

	line := 0
	for sc.Scan() {
		line++
		trimmed := bytes.TrimSpace(sc.Bytes())
		if len(trimmed) == 0 {
			continue // ignore blank lines
		}
		rec, err := ParseRecord(trimmed)
		if err != nil {
			if strict {
				return nil, fmt.Errorf("line %d: %v", line, err)
			}
			res.Skipped++
			continue
		}
		res.Records = append(res.Records, rec)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return res, nil
}
