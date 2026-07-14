package logq

// Head returns the first n records from records, preserving input order. When n
// is at least len(records) the whole slice is returned; when n is zero or
// negative the result is empty. The returned slice aliases the input's backing
// array and must not be mutated. Callers that treat n <= 0 as a user error
// (as `logq head` does) should reject it before calling.
func Head(records []*Record, n int) []*Record {
	if n <= 0 {
		return nil
	}
	if n >= len(records) {
		return records
	}
	return records[:n]
}
