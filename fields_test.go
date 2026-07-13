package logq

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestFields(t *testing.T) {
	// Values use json.Number, matching what the reader produces.
	recs := []*Record{
		NewRecord().Set("level", "info").Set("code", json.Number("200")).Set("msg", "a"),
		NewRecord().Set("level", "error").Set("code", "oops"), // code seen as both number and string
		NewRecord().Set("level", "info").Set("latency", json.Number("12.5")),
	}
	got := Fields(recs)
	want := []FieldInfo{
		{Name: "level", Types: []string{"string"}, Count: 3},
		{Name: "code", Types: []string{"number", "string"}, Count: 2},
		{Name: "msg", Types: []string{"string"}, Count: 1},
		{Name: "latency", Types: []string{"number"}, Count: 1},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Fields =\n  %+v\nwant\n  %+v", got, want)
	}
}

func TestFieldsEmpty(t *testing.T) {
	if got := Fields(nil); len(got) != 0 {
		t.Errorf("Fields(nil) = %v, want empty", got)
	}
}
