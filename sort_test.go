package logq

import (
	"encoding/json"
	"reflect"
	"testing"
)

// order extracts the field value from each record so a sorted result can be
// asserted compactly.
func order(recs []*Record, field string) []any {
	out := make([]any, len(recs))
	for i, r := range recs {
		v, _ := r.Get(field)
		out[i] = v
	}
	return out
}

func TestSortNumbersNumerically(t *testing.T) {
	// Bytewise these would be "10" < "100" < "2"; numerically 2 < 10 < 100.
	recs := []*Record{
		NewRecord().Set("n", json.Number("10")),
		NewRecord().Set("n", json.Number("100")),
		NewRecord().Set("n", json.Number("2")),
	}
	got := order(Sort(recs, "n", false), "n")
	want := []any{json.Number("2"), json.Number("10"), json.Number("100")}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ascending = %v, want %v", got, want)
	}
}

func TestSortStringsByCanonicalJSON(t *testing.T) {
	recs := []*Record{
		NewRecord().Set("s", "banana"),
		NewRecord().Set("s", "apple"),
		NewRecord().Set("s", "cherry"),
	}
	got := order(Sort(recs, "s", false), "s")
	want := []any{"apple", "banana", "cherry"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ascending = %v, want %v", got, want)
	}
}

// A number against a non-number pair compares by canonical JSON rendering,
// bytewise: "1" (0x22) < 1 (0x31) < null (0x6e) < true (0x74).
func TestSortMixedTypesByCanonicalJSON(t *testing.T) {
	recs := []*Record{
		NewRecord().Set("v", true),
		NewRecord().Set("v", json.Number("1")),
		NewRecord().Set("v", nil),
		NewRecord().Set("v", "1"),
	}
	got := order(Sort(recs, "v", false), "v")
	want := []any{"1", json.Number("1"), nil, true}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ascending = %v, want %v", got, want)
	}
}

func TestSortMissingFieldSortsLastPreservingInputOrder(t *testing.T) {
	recs := []*Record{
		NewRecord().Set("k", json.Number("3")),
		NewRecord().Set("other", "m1"), // missing k
		NewRecord().Set("k", json.Number("1")),
		NewRecord().Set("other", "m2"), // missing k
	}
	// Ascending: present sorted (1, 3), then the two missing in input order.
	got := Sort(recs, "k", false)
	if gv := order(got, "k"); !reflect.DeepEqual(gv[:2], []any{json.Number("1"), json.Number("3")}) {
		t.Errorf("present order = %v, want [1 3]", gv[:2])
	}
	if m := order(got[2:], "other"); !reflect.DeepEqual(m, []any{"m1", "m2"}) {
		t.Errorf("missing order = %v, want [m1 m2]", m)
	}

	// Descending reverses only the present comparison; missing still sort last
	// and keep input order.
	gotD := Sort(recs, "k", true)
	if gv := order(gotD, "k"); !reflect.DeepEqual(gv[:2], []any{json.Number("3"), json.Number("1")}) {
		t.Errorf("desc present order = %v, want [3 1]", gv[:2])
	}
	if m := order(gotD[2:], "other"); !reflect.DeepEqual(m, []any{"m1", "m2"}) {
		t.Errorf("desc missing order = %v, want [m1 m2]", m)
	}
}

// Ties keep input order in both directions; the tag field distinguishes records
// that share a sort value.
func TestSortIsStableForTies(t *testing.T) {
	recs := []*Record{
		NewRecord().Set("k", json.Number("1")).Set("tag", "a"),
		NewRecord().Set("k", json.Number("1")).Set("tag", "b"),
		NewRecord().Set("k", json.Number("1")).Set("tag", "c"),
	}
	if got := order(Sort(recs, "k", false), "tag"); !reflect.DeepEqual(got, []any{"a", "b", "c"}) {
		t.Errorf("ascending tie order = %v, want [a b c]", got)
	}
	if got := order(Sort(recs, "k", true), "tag"); !reflect.DeepEqual(got, []any{"a", "b", "c"}) {
		t.Errorf("descending tie order = %v, want [a b c]", got)
	}
}

func TestSortDescReversesComparison(t *testing.T) {
	recs := []*Record{
		NewRecord().Set("n", json.Number("2")),
		NewRecord().Set("n", json.Number("10")),
		NewRecord().Set("n", json.Number("100")),
	}
	got := order(Sort(recs, "n", true), "n")
	want := []any{json.Number("100"), json.Number("10"), json.Number("2")}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("descending = %v, want %v", got, want)
	}
}

// Sort must not mutate the caller's slice.
func TestSortDoesNotMutateInput(t *testing.T) {
	recs := []*Record{
		NewRecord().Set("n", json.Number("2")),
		NewRecord().Set("n", json.Number("1")),
	}
	before := order(recs, "n")
	Sort(recs, "n", false)
	if after := order(recs, "n"); !reflect.DeepEqual(after, before) {
		t.Errorf("input mutated: %v, was %v", after, before)
	}
}

// Same input => same output, independent of run count.
func TestSortIsDeterministic(t *testing.T) {
	recs := []*Record{
		NewRecord().Set("v", "b"),
		NewRecord().Set("v", json.Number("1")),
		NewRecord().Set("v", "a"),
		NewRecord().Set("v", nil),
	}
	first := order(Sort(recs, "v", false), "v")
	for i := 0; i < 5; i++ {
		if got := order(Sort(recs, "v", false), "v"); !reflect.DeepEqual(got, first) {
			t.Fatalf("run %d differs: %v vs %v", i, got, first)
		}
	}
}
