package logq

import "strings"

// Resolve looks up field in rec, supporting dot-separated nested access. It is
// the single field-resolution helper shared by every field-taking position
// (filter predicates, stats --group-by/--field, distinct, sort --by) so their
// lookup semantics never drift.
//
// An EXACT top-level key always wins: if rec has a key equal to the whole
// field string (dots and all), its value is returned without any traversal.
// Only when no such top-level key exists and field contains a "." is field
// split on "." and walked segment by segment — the first segment selects a
// top-level key and each subsequent segment indexes into the JSON object
// reached so far.
//
// Traversal resolves to "missing" — returning (nil, false), the same signal as
// a plain absent key — when an intermediate value is not a JSON object
// (map[string]any) or when a segment is absent. This mirrors Get's contract, so
// every consuming subcommand treats a missing nested path exactly as it treats
// a missing top-level field.
//
// Resolution is read-only and never mutates rec.
func (r *Record) Resolve(field string) (any, bool) {
	// Exact top-level key precedence: a literal key such as "a.b" is returned
	// as-is and is never reinterpreted as a nested path.
	if v, ok := r.Get(field); ok {
		return v, true
	}
	if !strings.Contains(field, ".") {
		return nil, false
	}

	segments := strings.Split(field, ".")
	cur, ok := r.Get(segments[0])
	if !ok {
		return nil, false
	}
	for _, seg := range segments[1:] {
		obj, ok := cur.(map[string]any)
		if !ok {
			return nil, false // traversal through a non-object value
		}
		cur, ok = obj[seg]
		if !ok {
			return nil, false // absent nested segment
		}
	}
	return cur, true
}
