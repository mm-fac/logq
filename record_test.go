package logq

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestParseRecordOrderAndValues(t *testing.T) {
	rec, err := ParseRecord([]byte(`{"ts":"t0","level":"info","code":200,"ok":true,"note":null}`))
	if err != nil {
		t.Fatalf("ParseRecord: %v", err)
	}
	wantKeys := []string{"ts", "level", "code", "ok", "note"}
	if got := rec.Keys(); !reflect.DeepEqual(got, wantKeys) {
		t.Errorf("keys = %v, want %v", got, wantKeys)
	}
	if v, _ := rec.Get("code"); v != json.Number("200") {
		t.Errorf("code = %#v, want json.Number(200)", v)
	}
	if v, _ := rec.Get("ok"); v != true {
		t.Errorf("ok = %#v, want true", v)
	}
	if v, ok := rec.Get("note"); v != nil || !ok {
		t.Errorf("note = %#v present=%v, want nil present=true", v, ok)
	}
	if _, ok := rec.Get("missing"); ok {
		t.Errorf("missing key reported present")
	}
}

func TestParseRecordDuplicateKeyLastWins(t *testing.T) {
	rec, err := ParseRecord([]byte(`{"a":1,"b":2,"a":3}`))
	if err != nil {
		t.Fatalf("ParseRecord: %v", err)
	}
	if got, want := rec.Keys(), []string{"a", "b"}; !reflect.DeepEqual(got, want) {
		t.Errorf("keys = %v, want %v", got, want)
	}
	if v, _ := rec.Get("a"); v != json.Number("3") {
		t.Errorf("a = %#v, want 3 (last wins)", v)
	}
}

func TestParseRecordRejects(t *testing.T) {
	cases := map[string]string{
		"array":        `[1,2,3]`,
		"scalar":       `42`,
		"string":       `"hello"`,
		"garbage":      `not json`,
		"empty":        ``,
		"trailing":     `{"a":1} {"b":2}`,
		"trailingjunk": `{"a":1}x`,
		"unterminated": `{"a":1`,
	}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			if rec, err := ParseRecord([]byte(in)); err == nil {
				t.Errorf("ParseRecord(%q) = %v, want error", in, rec)
			}
		})
	}
}

func TestTypeName(t *testing.T) {
	cases := []struct {
		in   any
		want string
	}{
		{nil, "null"},
		{true, "bool"},
		{json.Number("1"), "number"},
		{float64(1), "number"},
		{"s", "string"},
		{[]any{1}, "array"},
		{map[string]any{"a": 1}, "object"},
	}
	for _, c := range cases {
		if got := TypeName(c.in); got != c.want {
			t.Errorf("TypeName(%#v) = %q, want %q", c.in, got, c.want)
		}
	}
}
