package logq

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// predicateOps are the comparison operators accepted in a predicate, listed
// longest-first so that at any scan position a two-character operator is
// preferred over its one-character prefix (e.g. ">=" beats ">").
var predicateOps = []string{"==", "!=", ">=", "<=", ">", "<", "~"}

// Predicate is a parsed `field OP value` filter term. A record matches it when
// the record has Field and the comparison holds. Value typing follows the
// literal (requirements Q3): a value that parses as a number compares
// numerically, otherwise the comparison is on the value's string form; "~"
// (substring contains) is always string-based. A record missing Field never
// matches, and a field whose value cannot be interpreted in the inferred mode
// (e.g. a non-numeric value against a numeric literal) does not match either.
type Predicate struct {
	Field string
	Op    string
	Value string

	numeric bool    // Value parsed as a finite number and Op is not "~"
	num     float64 // parsed numeric value, valid when numeric is true
}

// ParsePredicate parses a single `field OP value` predicate. Surrounding
// whitespace around the field and value is ignored, so both "level==info" and
// "level == info" are accepted. It returns an error (suitable for a usage
// message) when no operator is present or the field or value is empty.
func ParsePredicate(s string) (Predicate, error) {
	idx, op := findOperator(s)
	if idx < 0 {
		return Predicate{}, fmt.Errorf("invalid predicate %q: expected field OP value with OP one of == != > >= < <= ~", s)
	}
	field := strings.TrimSpace(s[:idx])
	value := strings.TrimSpace(s[idx+len(op):])
	if field == "" {
		return Predicate{}, fmt.Errorf("invalid predicate %q: empty field name", s)
	}
	if value == "" {
		return Predicate{}, fmt.Errorf("invalid predicate %q: empty value", s)
	}

	p := Predicate{Field: field, Op: op, Value: value}
	if op != "~" {
		// Infer numeric comparison from the literal. NaN/Inf are excluded so a
		// value spelled "NaN"/"Inf" stays a plain string comparison.
		if f, err := strconv.ParseFloat(value, 64); err == nil && !math.IsInf(f, 0) && !math.IsNaN(f) {
			p.numeric = true
			p.num = f
		}
	}
	return p, nil
}

// findOperator returns the index and text of the first comparison operator in
// s, or (-1, "") if none is present. Two-character operators take precedence
// over one-character ones at the same position (see predicateOps).
func findOperator(s string) (int, string) {
	for i := 0; i < len(s); i++ {
		for _, op := range predicateOps {
			if strings.HasPrefix(s[i:], op) {
				return i, op
			}
		}
	}
	return -1, ""
}

// HasOperator reports whether s contains a recognized comparison operator. The
// CLI uses it to tell a predicate argument apart from an input file path.
func HasOperator(s string) bool {
	idx, _ := findOperator(s)
	return idx >= 0
}

// Match reports whether rec satisfies the predicate.
func (p Predicate) Match(rec *Record) bool {
	v, ok := rec.Resolve(p.Field)
	if !ok {
		return false // a record missing the field never matches
	}
	if p.Op == "~" {
		return strings.Contains(cellString(v), p.Value)
	}
	if p.numeric {
		rv, ok := numericValue(v)
		if !ok {
			return false // non-numeric value can't compare to a numeric literal
		}
		return compareNumbers(rv, p.num, p.Op)
	}
	return compareStrings(cellString(v), p.Value, p.Op)
}

// Filter returns the records satisfying every predicate (logical AND), in input
// order. With no predicates it returns all records.
func Filter(records []*Record, preds []Predicate) []*Record {
	out := make([]*Record, 0, len(records))
	for _, rec := range records {
		if matchesAll(rec, preds) {
			out = append(out, rec)
		}
	}
	return out
}

func matchesAll(rec *Record, preds []Predicate) bool {
	for _, p := range preds {
		if !p.Match(rec) {
			return false
		}
	}
	return true
}

// numericValue extracts a float64 from a JSON number value. Non-number values
// (strings, bools, null, arrays, objects) are not numerically comparable.
func numericValue(v any) (float64, bool) {
	switch t := v.(type) {
	case json.Number:
		f, err := t.Float64()
		if err != nil {
			return 0, false
		}
		return f, true
	case float64:
		return t, true
	default:
		return 0, false
	}
}

func compareNumbers(a, b float64, op string) bool {
	switch op {
	case "==":
		return a == b
	case "!=":
		return a != b
	case ">":
		return a > b
	case ">=":
		return a >= b
	case "<":
		return a < b
	case "<=":
		return a <= b
	}
	return false
}

func compareStrings(a, b, op string) bool {
	switch op {
	case "==":
		return a == b
	case "!=":
		return a != b
	case ">":
		return a > b
	case ">=":
		return a >= b
	case "<":
		return a < b
	case "<=":
		return a <= b
	}
	return false
}
