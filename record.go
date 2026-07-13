package logq

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// Record is one log entry: an ordered set of top-level key/value pairs.
//
// Key order is preserved as first-seen so that table/logfmt/json output is
// stable and readable (e.g. a leading "ts" stays leading). Values use the
// standard encoding/json Go types, with json.Number for numbers so digits are
// preserved exactly. v0 treats records as opaque top-level maps; nested
// objects/arrays are stored as-is but not indexed (no dotted-path access).
type Record struct {
	keys []string
	vals map[string]any
}

// NewRecord returns an empty Record ready for Set. It is used both by the
// reader and by subcommands that assemble output rows.
func NewRecord() *Record {
	return &Record{vals: make(map[string]any)}
}

// Set adds or replaces a key. First insertion fixes the key's position in the
// order; a later Set of the same key overwrites the value but keeps the
// position (last-value-wins, matching encoding/json for duplicate keys). It
// returns the receiver so calls can be chained.
func (r *Record) Set(key string, val any) *Record {
	if _, ok := r.vals[key]; !ok {
		r.keys = append(r.keys, key)
	}
	r.vals[key] = val
	return r
}

// Keys returns the keys in first-seen order. The slice is owned by the Record;
// callers must not mutate it.
func (r *Record) Keys() []string { return r.keys }

// Get returns the value for key and whether it was present.
func (r *Record) Get(key string) (any, bool) {
	v, ok := r.vals[key]
	return v, ok
}

// Len reports the number of keys.
func (r *Record) Len() int { return len(r.keys) }

// ParseRecord parses a single JSON object into a Record, preserving key order.
// It returns an error if the input is not exactly one JSON object (arrays,
// scalars, or trailing data are rejected), which the reader treats as a
// malformed line.
func ParseRecord(data []byte) (*Record, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()

	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		return nil, fmt.Errorf("not a JSON object")
	}

	rec := NewRecord()
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, ok := keyTok.(string)
		if !ok {
			return nil, fmt.Errorf("expected string key, got %v", keyTok)
		}
		var val any
		if err := dec.Decode(&val); err != nil {
			return nil, err
		}
		rec.Set(key, val)
	}

	// Consume the closing '}'.
	if _, err := dec.Token(); err != nil {
		return nil, err
	}
	// Reject anything after the object (a line must be exactly one object).
	if _, err := dec.Token(); err != io.EOF {
		return nil, fmt.Errorf("trailing data after JSON object")
	}
	return rec, nil
}

// TypeName reports the JSON value type of v as one of:
// "null", "bool", "number", "string", "array", "object".
func TypeName(v any) string {
	switch v.(type) {
	case nil:
		return "null"
	case bool:
		return "bool"
	case json.Number, float64:
		return "number"
	case string:
		return "string"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	default:
		return fmt.Sprintf("%T", v)
	}
}
