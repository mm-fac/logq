package logq

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestDistinctCountsSortedByCanonicalJSON(t *testing.T) {
	recs := []*Record{
		NewRecord().Set("level", "info"),
		NewRecord().Set("level", "warn"),
		NewRecord().Set("level", "info"),
		NewRecord().Set("level", "error"),
		NewRecord().Set("level", "info"),
	}
	got, missing := Distinct(recs, "level")
	if missing != 0 {
		t.Errorf("missing = %d, want 0", missing)
	}
	want := []DistinctValue{
		{Value: "error", Count: 1},
		{Value: "info", Count: 3},
		{Value: "warn", Count: 1},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Distinct =\n  %+v\nwant\n  %+v", got, want)
	}
}

// Different JSON types that render alike must stay distinct, and rows must be
// ordered by each value's canonical JSON rendering (bytewise): "1" (0x22) < 1
// (0x31) < null (0x6e) < true (0x74).
func TestDistinctSeparatesValuesOfDifferentTypes(t *testing.T) {
	recs := []*Record{
		NewRecord().Set("v", json.Number("1")),
		NewRecord().Set("v", "1"),
		NewRecord().Set("v", json.Number("1")),
		NewRecord().Set("v", true),
		NewRecord().Set("v", nil),
	}
	got, missing := Distinct(recs, "v")
	if missing != 0 {
		t.Errorf("missing = %d, want 0", missing)
	}
	want := []DistinctValue{
		{Value: "1", Count: 1},
		{Value: json.Number("1"), Count: 2},
		{Value: nil, Count: 1},
		{Value: true, Count: 1},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Distinct =\n  %+v\nwant\n  %+v", got, want)
	}
}

func TestDistinctReportsMissingCount(t *testing.T) {
	recs := []*Record{
		NewRecord().Set("a", "x"),
		NewRecord().Set("b", "y"), // missing a
		NewRecord().Set("a", "x"),
		NewRecord().Set("c", "z"), // missing a
	}
	got, missing := Distinct(recs, "a")
	if missing != 2 {
		t.Errorf("missing = %d, want 2", missing)
	}
	want := []DistinctValue{{Value: "x", Count: 2}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Distinct = %+v, want %+v", got, want)
	}
}

// Ordering is a stable function of the values, independent of input order and
// of Go's map iteration order.
func TestDistinctOrderingIsDeterministic(t *testing.T) {
	vals := []string{"c", "a", "b", "a", "c", "a"}
	recs := make([]*Record, len(vals))
	for i, v := range vals {
		recs[i] = NewRecord().Set("k", v)
	}
	first, _ := Distinct(recs, "k")
	for i := 0; i < 5; i++ {
		got, _ := Distinct(recs, "k")
		if !reflect.DeepEqual(got, first) {
			t.Fatalf("run %d differs:\n %+v\nvs\n %+v", i, got, first)
		}
	}
	want := []DistinctValue{
		{Value: "a", Count: 3},
		{Value: "b", Count: 1},
		{Value: "c", Count: 2},
	}
	if !reflect.DeepEqual(first, want) {
		t.Errorf("Distinct = %+v, want %+v", first, want)
	}
}
