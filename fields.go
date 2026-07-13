package logq

import "sort"

// FieldInfo summarizes one field key observed across a set of records.
type FieldInfo struct {
	// Name is the field key.
	Name string
	// Types are the distinct JSON value types observed for this key, sorted
	// for deterministic output (see TypeName for the vocabulary).
	Types []string
	// Count is the number of records that contained this key.
	Count int
}

// Fields computes the per-key summary that backs `logq fields`: for every key
// seen across records, its distinct value types and how many records contained
// it. Keys are returned in first-seen order across the input.
func Fields(records []*Record) []FieldInfo {
	var order []string
	types := make(map[string]map[string]bool)
	counts := make(map[string]int)

	for _, rec := range records {
		for _, k := range rec.Keys() {
			if _, seen := counts[k]; !seen {
				order = append(order, k)
				types[k] = make(map[string]bool)
			}
			counts[k]++
			v, _ := rec.Get(k)
			types[k][TypeName(v)] = true
		}
	}

	out := make([]FieldInfo, 0, len(order))
	for _, k := range order {
		ts := make([]string, 0, len(types[k]))
		for t := range types[k] {
			ts = append(ts, t)
		}
		sort.Strings(ts)
		out = append(out, FieldInfo{Name: k, Types: ts, Count: counts[k]})
	}
	return out
}
