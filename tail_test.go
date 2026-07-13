package logq

import (
	"reflect"
	"testing"
)

func TestTail(t *testing.T) {
	mk := func(ids ...string) []*Record {
		recs := make([]*Record, len(ids))
		for i, id := range ids {
			recs[i] = NewRecord().Set("id", id)
		}
		return recs
	}
	base := mk("1", "2", "3")

	tests := []struct {
		name string
		recs []*Record
		n    int
		want []string
	}{
		{"fewer than count", base, 2, []string{"2", "3"}},
		{"exact count", base, 3, []string{"1", "2", "3"}},
		{"n larger than count", base, 10, []string{"1", "2", "3"}},
		{"single", base, 1, []string{"3"}},
		{"empty input", nil, 5, nil},
		{"n zero", base, 0, nil},
		{"n negative", base, -4, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Tail(tt.recs, tt.n)
			var ids []string
			for _, r := range got {
				v, _ := r.Get("id")
				ids = append(ids, v.(string))
			}
			if !reflect.DeepEqual(ids, tt.want) {
				t.Errorf("Tail(_, %d) ids = %v, want %v", tt.n, ids, tt.want)
			}
		})
	}
}
