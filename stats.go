package logq

import (
	"encoding/json"
	"fmt"
	"sort"
)

// StatsGroup summarizes records sharing one group-by field value.
type StatsGroup struct {
	// Value is the group-by field value.
	Value any
	// Count is the number of records in the group.
	Count int
	// NumericCount is the number of numeric values aggregated for Field.
	NumericCount int
	// Skipped is the number of records whose Field value was missing or non-numeric.
	Skipped int
	Min     float64
	Max     float64
	Sum     float64
	Avg     float64
}

// Stats groups records by groupBy and, when field is non-empty, aggregates the
// numeric values of field per group. Groups are returned in deterministic
// order by their canonical key.
func Stats(records []*Record, groupBy, field string) []StatsGroup {
	type acc struct {
		group StatsGroup
		key   string
	}

	var order []string
	groups := make(map[string]*acc)
	for _, rec := range records {
		v, ok := rec.Get(groupBy)
		if !ok {
			v = nil
		}
		key := groupKey(v)
		g, ok := groups[key]
		if !ok {
			g = &acc{group: StatsGroup{Value: v}, key: key}
			groups[key] = g
			order = append(order, key)
		}
		g.group.Count++

		if field == "" {
			continue
		}
		fv, ok := rec.Get(field)
		n, numeric := numberValue(fv)
		if !ok || !numeric {
			g.group.Skipped++
			continue
		}
		if g.group.NumericCount == 0 {
			g.group.Min = n
			g.group.Max = n
		} else {
			if n < g.group.Min {
				g.group.Min = n
			}
			if n > g.group.Max {
				g.group.Max = n
			}
		}
		g.group.NumericCount++
		g.group.Sum += n
		g.group.Avg = g.group.Sum / float64(g.group.NumericCount)
	}

	sort.Strings(order)
	out := make([]StatsGroup, 0, len(order))
	for _, key := range order {
		out = append(out, groups[key].group)
	}
	return out
}

func groupKey(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%T:%v", v, v)
	}
	return fmt.Sprintf("%s:%s", TypeName(v), b)
}

func numberValue(v any) (float64, bool) {
	switch n := v.(type) {
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	case float64:
		return n, true
	default:
		return 0, false
	}
}
