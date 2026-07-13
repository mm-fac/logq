package logq

// Tail returns the last n records from records, preserving input order. When n
// is at least len(records) the whole slice is returned; when n is zero or
// negative the result is empty. The returned slice aliases the input's backing
// array and must not be mutated. Callers that treat n <= 0 as a user error
// (as `logq tail` does) should reject it before calling.
func Tail(records []*Record, n int) []*Record {
	if n <= 0 {
		return nil
	}
	if n >= len(records) {
		return records
	}
	return records[len(records)-n:]
}
