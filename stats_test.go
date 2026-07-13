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
		{Value: "error", Count: 2, NumericCount: 1, Skipped: 1, Min: 30, Max: 30, Sum: 30, Avg: 30},
		{Value: "info", Count: 3, NumericCount: 2, Skipped: 1, Min: 7.5, Max: 12.5, Sum: 20, Avg: 10},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Stats aggregations =\n  %+v\nwant\n  %+v", got, want)
	}
}
