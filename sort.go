package logq

import (
	"encoding/json"
	"math/big"
	"sort"
)

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
	// When both sides are JSON number literals, compare them exactly via
	// math/big.Rat so ordering is correct for every valid JSON number regardless
	// of precision or magnitude — no float64 round-trip, no overflow fallback.
	if an, ok := a.(json.Number); ok {
		if bn, ok := b.(json.Number); ok {
			if c, exact := compareJSONNumbers(an, bn); exact {
				return c < 0
			}
			// A literal Rat cannot parse (should never happen for valid JSON):
			// fail safe to the canonical-JSON fallback for this pair.
			return canonicalJSON(a) < canonicalJSON(b)
		}
	}
	// Any other pair — including a number against a non-number — compares by
	// canonical JSON rendering, bytewise.
	return canonicalJSON(a) < canonicalJSON(b)
}

// compareJSONNumbers returns the sign of a-b as exact rationals, and whether the
// comparison was exact. math/big.Rat.SetString parses decimal literals with
// e-exponents exactly, so any valid JSON number compares precisely; exact is
// false only if a literal fails to parse, letting the caller fall back.
func compareJSONNumbers(a, b json.Number) (int, bool) {
	ar, aok := new(big.Rat).SetString(a.String())
	br, bok := new(big.Rat).SetString(b.String())
	if !aok || !bok {
		return 0, false
	}
	return ar.Cmp(br), true
}
