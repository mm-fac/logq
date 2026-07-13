package logq

import "sort"

// Sort orders records by the top-level field and returns a new slice; the input
// slice is left untouched.
//
// Comparison rules:
//   - Two JSON-number values compare numerically. Any other pair (including a
//     number against a non-number) compares by canonical JSON rendering,
//     bytewise — the same identity distinct uses, so different JSON types never
//     collide even when they render alike (the number 1 vs the string "1").
//   - Records that do not contain field sort LAST, after every record that has
//     it, and keep their relative input order among themselves. This holds
//     regardless of desc.
//   - The sort is stable: records that compare equal keep their input order, so
//     the same input always yields the same output.
//
// When desc is true the comparison of present records is reversed, but the
// missing-last placement and the input order among missing (and among ties) are
// not.
func Sort(records []*Record, field string, desc bool) []*Record {
	present := make([]*Record, 0, len(records))
	var missing []*Record
	for _, rec := range records {
		if _, ok := rec.Get(field); ok {
			present = append(present, rec)
		} else {
			missing = append(missing, rec)
		}
	}

	sort.SliceStable(present, func(i, j int) bool {
		vi, _ := present[i].Get(field)
		vj, _ := present[j].Get(field)
		if desc {
			return lessValue(vj, vi)
		}
		return lessValue(vi, vj)
	})

	return append(present, missing...)
}

// lessValue reports whether a sorts before b under the field-comparison rules:
// numeric when both values are JSON numbers, otherwise bytewise on the canonical
// JSON rendering.
func lessValue(a, b any) bool {
	an, aok := numericValue(a)
	bn, bok := numericValue(b)
	if aok && bok {
		return an < bn
	}
	return canonicalJSON(a) < canonicalJSON(b)
}
