package logq

import (
	"encoding/json"
	"reflect"
	"testing"
)

// nested builds a record shaped like {"user":{"role":"admin","meta":{"id":7}}}
// so traversal can be exercised several levels deep.
func nestedUserRecord() *Record {
	return NewRecord().Set("user", map[string]any{
		"role": "admin",
		"meta": map[string]any{"id": json.Number("7")},
	})
}

func TestResolve(t *testing.T) {
	tests := []struct {
		name  string
		rec   *Record
		field string
		want  any
		wxist bool
	}{
		{
			name:  "plain top-level key",
			rec:   NewRecord().Set("level", "info"),
			field: "level",
			want:  "info",
			wxist: true,
		},
		{
			name:  "one-level traversal",
			rec:   nestedUserRecord(),
			field: "user.role",
			want:  "admin",
			wxist: true,
		},
		{
			name:  "multi-level traversal",
			rec:   nestedUserRecord(),
			field: "user.meta.id",
			want:  json.Number("7"),
			wxist: true,
		},
		{
			// An exact top-level key "user.role" wins over the nested path that
			// would otherwise resolve to "admin".
			name:  "exact top-level key beats nested path",
			rec:   NewRecord().Set("user.role", "literal").Set("user", map[string]any{"role": "admin"}),
			field: "user.role",
			want:  "literal",
			wxist: true,
		},
		{
			// The exact-key check applies to the whole dotted string, so a bare
			// literal dotted key resolves even with no nested object present.
			name:  "literal dotted key with no nested object",
			rec:   NewRecord().Set("a.b", json.Number("1")),
			field: "a.b",
			want:  json.Number("1"),
			wxist: true,
		},
		{
			name:  "traversal through a non-object intermediate is missing",
			rec:   NewRecord().Set("a", "scalar"),
			field: "a.b",
			want:  nil,
			wxist: false,
		},
		{
			name:  "traversal through a number intermediate is missing",
			rec:   NewRecord().Set("a", map[string]any{"b": json.Number("1")}),
			field: "a.b.c",
			want:  nil,
			wxist: false,
		},
		{
			name:  "absent nested segment is missing",
			rec:   nestedUserRecord(),
			field: "user.absent",
			want:  nil,
			wxist: false,
		},
		{
			name:  "absent first segment is missing",
			rec:   nestedUserRecord(),
			field: "nope.role",
			want:  nil,
			wxist: false,
		},
		{
			name:  "dotted lookup on absent plain field is missing",
			rec:   NewRecord().Set("level", "info"),
			field: "user.role",
			want:  nil,
			wxist: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := tc.rec.Resolve(tc.field)
			if ok != tc.wxist {
				t.Fatalf("Resolve(%q) ok = %v, want %v", tc.field, ok, tc.wxist)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Resolve(%q) = %#v, want %#v", tc.field, got, tc.want)
			}
		})
	}
}

// Resolve must not mutate the record it reads (no key added by a failed lookup).
func TestResolveDoesNotMutate(t *testing.T) {
	rec := nestedUserRecord()
	before := append([]string(nil), rec.Keys()...)
	rec.Resolve("user.meta.id")
	rec.Resolve("user.absent")
	rec.Resolve("nope.role")
	if !reflect.DeepEqual(rec.Keys(), before) {
		t.Errorf("keys changed after Resolve: %v, want %v", rec.Keys(), before)
	}
}

// Each of the four field-taking positions must resolve a nested path.

func TestFilterNestedPredicate(t *testing.T) {
	recs := []*Record{
		NewRecord().Set("user", map[string]any{"role": "admin"}),
		NewRecord().Set("user", map[string]any{"role": "guest"}),
		NewRecord().Set("user", "scalar"), // non-object intermediate -> missing -> no match
		NewRecord().Set("other", "x"),     // absent -> no match
	}
	p, err := ParsePredicate("user.role==admin")
	if err != nil {
		t.Fatalf("ParsePredicate: %v", err)
	}
	got := Filter(recs, []Predicate{p})
	if len(got) != 1 {
		t.Fatalf("matched %d records, want 1", len(got))
	}
	if v, _ := got[0].Resolve("user.role"); v != "admin" {
		t.Errorf("matched record user.role = %v, want admin", v)
	}
}

func TestStatsNestedGroupByAndField(t *testing.T) {
	recs := []*Record{
		NewRecord().Set("user", map[string]any{"role": "admin"}).Set("req", map[string]any{"ms": json.Number("10")}),
		NewRecord().Set("user", map[string]any{"role": "admin"}).Set("req", map[string]any{"ms": json.Number("30")}),
		NewRecord().Set("user", map[string]any{"role": "guest"}).Set("req", map[string]any{"ms": json.Number("50")}),
		NewRecord().Set("user", map[string]any{"role": "guest"}).Set("req", "scalar"), // nested field missing -> skipped
	}
	groups := Stats(recs, "user.role", "req.ms")
	if len(groups) != 2 {
		t.Fatalf("got %d groups, want 2", len(groups))
	}
	// Groups are ordered by canonical key: "admin" before "guest".
	admin, guest := groups[0], groups[1]
	if admin.Value != "admin" || admin.Count != 2 || admin.NumericCount != 2 || admin.Sum != json.Number("40") || admin.Avg != 20 {
		t.Errorf("admin group = %+v", admin)
	}
	if guest.Value != "guest" || guest.Count != 2 || guest.NumericCount != 1 || guest.Skipped != 1 || guest.Sum != json.Number("50") {
		t.Errorf("guest group = %+v", guest)
	}
}

func TestDistinctNestedField(t *testing.T) {
	recs := []*Record{
		NewRecord().Set("user", map[string]any{"role": "admin"}),
		NewRecord().Set("user", map[string]any{"role": "guest"}),
		NewRecord().Set("user", map[string]any{"role": "admin"}),
		NewRecord().Set("user", "scalar"), // non-object intermediate -> missing
		NewRecord().Set("x", "y"),         // absent -> missing
	}
	got, missing := Distinct(recs, "user.role")
	if missing != 2 {
		t.Errorf("missing = %d, want 2", missing)
	}
	want := []DistinctValue{
		{Value: "admin", Count: 2},
		{Value: "guest", Count: 1},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Distinct =\n  %+v\nwant\n  %+v", got, want)
	}
}

func TestSortNestedField(t *testing.T) {
	missingRec := NewRecord().Set("user", "scalar") // non-object -> sorts last
	recs := []*Record{
		NewRecord().Set("user", map[string]any{"n": json.Number("100")}),
		missingRec,
		NewRecord().Set("user", map[string]any{"n": json.Number("2")}),
		NewRecord().Set("user", map[string]any{"n": json.Number("10")}),
	}
	got := Sort(recs, "user.n", false)
	gotNs := make([]any, len(got))
	for i, r := range got {
		v, _ := r.Resolve("user.n")
		gotNs[i] = v
	}
	// 2 < 10 < 100 numerically, then the non-object record last (nil).
	want := []any{json.Number("2"), json.Number("10"), json.Number("100"), nil}
	if !reflect.DeepEqual(gotNs, want) {
		t.Errorf("sorted user.n = %v, want %v", gotNs, want)
	}
	if got[3] != missingRec {
		t.Errorf("record with non-object intermediate did not sort last")
	}
}

// The exact-key precedence rule must hold through the consuming subcommands, not
// just the helper in isolation: a literal dotted key is grouped by its own value.
func TestStatsExactKeyPrecedence(t *testing.T) {
	recs := []*Record{
		NewRecord().Set("user.role", "literal").Set("user", map[string]any{"role": "nested"}),
	}
	groups := Stats(recs, "user.role", "")
	if len(groups) != 1 {
		t.Fatalf("got %d groups, want 1", len(groups))
	}
	if groups[0].Value != "literal" {
		t.Errorf("group value = %v, want literal (exact top-level key wins)", groups[0].Value)
	}
}
