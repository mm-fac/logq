package logq

import (
	"encoding/json"
	"fmt"
	"sort"
)

// DistinctValue is one distinct value of a field together with the number of
// records in which that exact value occurred.
type DistinctValue struct {
	// Value is the field value, using the same Go types as the reader
	// (json.Number for numbers, string, bool, nil, []any, map[string]any).
	Value any
	// Count is the number of records whose field value shares this value's
	// canonical JSON rendering.
	Count int
}

// Distinct lists each distinct value of field across records with its
// occurrence count. Two values are the same distinct value iff they share the
// same canonical JSON rendering, so values of different JSON types stay
// distinct even when they display alike (the number 1 vs the string "1").
//
// Records that do not contain field are not represented; their number is
// returned as missing so the caller can report it via the standard
// skipped-count mechanism. Results are sorted ascending, bytewise, by each
// value's canonical JSON rendering.
func Distinct(records []*Record, field string) (values []DistinctValue, missing int) {
	var order []string
	counts := make(map[string]int)
	vals := make(map[string]any)
	for _, rec := range records {
		v, ok := rec.Resolve(field)
		if !ok {
			missing++
			continue
		}
		key := canonicalJSON(v)
		if _, seen := counts[key]; !seen {
			order = append(order, key)
			vals[key] = v
		}
		counts[key]++
	}

	sort.Strings(order)
	out := make([]DistinctValue, 0, len(order))
	for _, key := range order {
		out = append(out, DistinctValue{Value: vals[key], Count: counts[key]})
	}
	return out, missing
}

// TopN returns the n most frequent of the given distinct values, ordered by
// Count descending with ties broken by each value's canonical JSON rendering
// ascending, bytewise (the same rendering Distinct sorts by). It returns
// min(n, len(values)) values, or an empty slice when n <= 0. The input is left
// unmodified. The tie-break makes the order total, so equal inputs always yield
// equal output.
func TopN(values []DistinctValue, n int) []DistinctValue {
	if n <= 0 {
		return nil
	}
	sorted := make([]DistinctValue, len(values))
	copy(sorted, values)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Count != sorted[j].Count {
			return sorted[i].Count > sorted[j].Count
		}
		return canonicalJSON(sorted[i].Value) < canonicalJSON(sorted[j].Value)
	})
	if n > len(sorted) {
		n = len(sorted)
	}
	return sorted[:n]
}

// canonicalJSON renders v as its canonical JSON encoding, used both as a
// distinct value's identity and as its sort key. JSON encodes each type
// unambiguously (strings are quoted, null is bare, numbers keep their literal),
// so distinct types never collide even when they display alike.
func canonicalJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%T:%v", v, v)
	}
	return string(b)
}
