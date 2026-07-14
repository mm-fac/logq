package logq

import (
	"encoding/json"
	"math/big"
)

// CompareJSONNumbers compares two JSON-number literals exactly. It returns the
// sign of a-b (-1 if a<b, 0 if a==b, +1 if a>b) together with whether the
// comparison was exact.
//
// Both operands are parsed as arbitrary-precision rationals via math/big.Rat, so
// every valid JSON number compares precisely regardless of magnitude or
// fractional precision — no float64 round-trip that would collapse integers past
// 2^53 (e.g. 9007199254740993 vs 9007199254740992) or overflow large exponents
// to infinity (e.g. 10e400 vs 9e400). This mirrors the exact ordering `sort`
// uses over the raw literals.
//
// exact is false only when a literal fails to parse as a rational (which should
// not happen for a value produced by encoding/json's number decoding), letting
// callers fall back to an approximate comparison.
func CompareJSONNumbers(a, b json.Number) (sign int, exact bool) {
	ar, aok := new(big.Rat).SetString(a.String())
	br, bok := new(big.Rat).SetString(b.String())
	if !aok || !bok {
		return 0, false
	}
	return ar.Cmp(br), true
}
