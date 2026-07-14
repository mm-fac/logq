package logq

import (
	"encoding/json"
	"testing"
)

func TestCompareJSONNumbersExact(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"1", "2", -1},
		{"2", "1", 1},
		{"200", "200", 0},
		{"12.5", "12.50", 0}, // trailing-zero difference is numerically equal
		{"10", "2", 1},       // numeric, not bytewise ("10" < "2" lexically)

		// Integers past float64's 53-bit mantissa: 9007199254740993 and
		// 9007199254740992 both round to the same float64, so a float-based
		// comparison would call them equal. Exact comparison keeps them distinct.
		{"9007199254740993", "9007199254740992", 1},
		{"9007199254740992", "9007199254740993", -1},
		{"9007199254740993", "9007199254740993", 0},

		// Magnitudes past float64's range: 10e400 and 9e400 both overflow to +Inf,
		// so a float comparison sees them as equal. Exact comparison orders them.
		{"10e400", "9e400", 1},
		{"9e400", "10e400", -1},
		{"9e400", "9e400", 0},
	}
	for _, tt := range tests {
		got, exact := CompareJSONNumbers(json.Number(tt.a), json.Number(tt.b))
		if !exact {
			t.Errorf("CompareJSONNumbers(%q, %q) exact=false, want exact", tt.a, tt.b)
			continue
		}
		if got != tt.want {
			t.Errorf("CompareJSONNumbers(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestCompareJSONNumbersNonNumericIsNotExact(t *testing.T) {
	// A literal that is not a valid rational reports exact=false so callers can
	// fall back rather than trust a meaningless sign.
	if _, exact := CompareJSONNumbers(json.Number("nope"), json.Number("1")); exact {
		t.Errorf("CompareJSONNumbers(nope, 1) exact=true, want false")
	}
	if _, exact := CompareJSONNumbers(json.Number("1"), json.Number("")); exact {
		t.Errorf("CompareJSONNumbers(1, empty) exact=true, want false")
	}
}
