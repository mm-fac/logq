package logq

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestStatsGroupCountsSortedByGroupKey(t *testing.T) {
	recs := []*Record{
		NewRecord().Set("level", "info"),
		NewRecord().Set("level", "error"),
		NewRecord().Set("level", "info"),
	}
	got := Stats(recs, "level", "")
	want := []StatsGroup{
		{Value: "error", Count: 1},
		{Value: "info", Count: 2},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Stats counts =\n  %+v\nwant\n  %+v", got, want)
	}
}

func TestStatsAggregatesNumericFieldAndSkipsNonNumeric(t *testing.T) {
	recs := []*Record{
		NewRecord().Set("level", "info").Set("latency", json.Number("12.5")),
		NewRecord().Set("level", "error").Set("latency", json.Number("30")),
		NewRecord().Set("level", "info").Set("latency", json.Number("7.5")),
		NewRecord().Set("level", "info").Set("latency", "slow"),
		NewRecord().Set("level", "error").Set("latency", "n/a"),
	}
	got := Stats(recs, "level", "latency")
	want := []StatsGroup{
		{Value: "error", Count: 2, NumericCount: 1, Skipped: 1, Min: json.Number("30"), Max: json.Number("30"), Sum: json.Number("30"), Avg: 30},
		{Value: "info", Count: 3, NumericCount: 2, Skipped: 1, Min: json.Number("7.5"), Max: json.Number("12.5"), Sum: float64(20), Avg: 10},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Stats aggregations =\n  %+v\nwant\n  %+v", got, want)
	}
}

func TestStatsSelectsExactExtremaAndSumsIntegerLiteralsExactly(t *testing.T) {
	recs := []*Record{
		NewRecord().Set("group", "negative").Set("n", json.Number("-9007199254740992")),
		NewRecord().Set("group", "negative").Set("n", json.Number("-9007199254740993")),
		NewRecord().Set("group", "positive").Set("n", json.Number("9007199254740992")),
		NewRecord().Set("group", "positive").Set("n", json.Number("9007199254740993")),
	}

	got := Stats(recs, "group", "n")
	want := []StatsGroup{
		{
			Value: "negative", Count: 2, NumericCount: 2,
			Min: json.Number("-9007199254740993"), Max: json.Number("-9007199254740992"),
			Sum: json.Number("-18014398509481985"), Avg: -9007199254740992,
		},
		{
			Value: "positive", Count: 2, NumericCount: 2,
			Min: json.Number("9007199254740992"), Max: json.Number("9007199254740993"),
			Sum: json.Number("18014398509481985"), Avg: 9007199254740992,
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Stats exact aggregations =\n  %+v\nwant\n  %+v", got, want)
	}
}

func TestStatsFractionalValueFallsBackToFloat64Sum(t *testing.T) {
	recs := []*Record{
		NewRecord().Set("group", "all").Set("n", json.Number("9007199254740992")),
		NewRecord().Set("group", "all").Set("n", json.Number("0.5")),
	}

	got := Stats(recs, "group", "n")
	if len(got) != 1 {
		t.Fatalf("Stats group count = %d, want 1", len(got))
	}
	if sum, ok := got[0].Sum.(float64); !ok || sum != 9007199254740992 {
		t.Errorf("Stats sum = %v (%T), want float64 fallback 9007199254740992", got[0].Sum, got[0].Sum)
	}
}
