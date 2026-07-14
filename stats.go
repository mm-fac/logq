package logq

import (
	"encoding/json"
	"fmt"
	"math/big"
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
	// Min and Max preserve the selected json.Number literal. Programmatically
	// supplied float64 values remain float64.
	Min any
	Max any
	// Sum is a json.Number for all-integer-literal groups and a float64 otherwise.
	Sum any
	// Avg is intentionally an approximate float64 in v0.2.
	Avg float64
}

// Stats groups records by groupBy and, when field is non-empty, aggregates the
// numeric values of field per group. Groups are returned in deterministic
// order by their canonical key.
func Stats(records []*Record, groupBy, field string) []StatsGroup {
	type acc struct {
		group       StatsGroup
		key         string
		min         statsNumber
		max         statsNumber
		floatSum    float64
		integerSum  big.Int
		allIntegers bool
	}

	var order []string
	groups := make(map[string]*acc)
	for _, rec := range records {
		v, ok := rec.Resolve(groupBy)
		if !ok {
			v = nil
		}
		key := groupKey(v)
		g, ok := groups[key]
		if !ok {
			g = &acc{group: StatsGroup{Value: v}, key: key}
			if field != "" {
				// Preserve the existing aggregate output for groups with no
				// numeric values.
				g.group.Min = float64(0)
				g.group.Max = float64(0)
				g.group.Sum = float64(0)
			}
			groups[key] = g
			order = append(order, key)
		}
		g.group.Count++

		if field == "" {
			continue
		}
		fv, ok := rec.Resolve(field)
		n, numeric := statsNumberValue(fv)
		if !ok || !numeric {
			g.group.Skipped++
			continue
		}
		if g.group.NumericCount == 0 {
			g.min = n
			g.max = n
			g.group.Min = n.value
			g.group.Max = n.value
			if n.integer != nil {
				g.integerSum.Set(n.integer)
				g.allIntegers = true
			}
		} else {
			if compareStatsNumbers(n, g.min) < 0 {
				g.min = n
				g.group.Min = n.value
			}
			if compareStatsNumbers(n, g.max) > 0 {
				g.max = n
				g.group.Max = n.value
			}
			if g.allIntegers {
				if n.integer == nil {
					g.allIntegers = false
				} else {
					g.integerSum.Add(&g.integerSum, n.integer)
				}
			}
		}
		g.group.NumericCount++
		g.floatSum += n.approximate
		if g.allIntegers {
			g.group.Sum = json.Number(g.integerSum.String())
		} else {
			g.group.Sum = g.floatSum
		}
		g.group.Avg = g.floatSum / float64(g.group.NumericCount)
	}

	sort.Strings(order)
	out := make([]StatsGroup, 0, len(order))
	for _, key := range order {
		out = append(out, groups[key].group)
	}
	return out
}

type statsNumber struct {
	value       any
	literal     json.Number
	approximate float64
	integer     *big.Int
}

func statsNumberValue(v any) (statsNumber, bool) {
	switch n := v.(type) {
	case json.Number:
		f, err := n.Float64()
		if err != nil {
			return statsNumber{}, false
		}
		integer, _ := new(big.Int).SetString(n.String(), 10)
		return statsNumber{value: n, literal: n, approximate: f, integer: integer}, true
	case float64:
		return statsNumber{value: n, approximate: n}, true
	default:
		return statsNumber{}, false
	}
}

func compareStatsNumbers(a, b statsNumber) int {
	if a.literal != "" && b.literal != "" {
		if sign, exact := CompareJSONNumbers(a.literal, b.literal); exact {
			return sign
		}
	}
	if a.approximate < b.approximate {
		return -1
	}
	if a.approximate > b.approximate {
		return 1
	}
	return 0
}

func groupKey(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%T:%v", v, v)
	}
	return fmt.Sprintf("%s:%s", TypeName(v), b)
}
