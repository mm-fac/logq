package logq

import (
	"encoding/json"
	"testing"
)

func TestParsePredicate(t *testing.T) {
	tests := []struct {
		in        string
		wantField string
		wantOp    string
		wantValue string
		wantNum   bool
	}{
		{"level==info", "level", "==", "info", false},
		{"level == info", "level", "==", "info", false}, // whitespace tolerated
		{"code==200", "code", "==", "200", true},        // numeric literal
		{"code!=200", "code", "!=", "200", true},
		{"latency>=12.5", "latency", ">=", "12.5", true},
		{"latency<12.5", "latency", "<", "12.5", true},
		{"code>=200", "code", ">=", "200", true},
		{"code<=200", "code", "<=", "200", true},
		{"msg~oo", "msg", "~", "oo", false}, // ~ is never numeric
		{"n~5", "n", "~", "5", false},       // ~ with number literal stays string
		{"path==/var/log", "path", "==", "/var/log", false},
		{"NaN==NaN", "NaN", "==", "NaN", false}, // NaN literal is not treated as numeric
	}
	for _, tt := range tests {
		p, err := ParsePredicate(tt.in)
		if err != nil {
			t.Errorf("ParsePredicate(%q) error: %v", tt.in, err)
			continue
		}
		if p.Field != tt.wantField || p.Op != tt.wantOp || p.Value != tt.wantValue || p.numeric != tt.wantNum {
			t.Errorf("ParsePredicate(%q) = {%q %q %q numeric=%v}, want {%q %q %q numeric=%v}",
				tt.in, p.Field, p.Op, p.Value, p.numeric, tt.wantField, tt.wantOp, tt.wantValue, tt.wantNum)
		}
	}
}

func TestParsePredicateErrors(t *testing.T) {
	bad := []string{
		"",            // empty
		"level",       // no operator
		"level error", // no operator (space is not an operator)
		"==info",      // empty field
		"level==",     // empty value
		"  ==  ",      // empty field and value
		"level=info",  // single '=' is not a recognized operator
	}
	for _, in := range bad {
		if _, err := ParsePredicate(in); err == nil {
			t.Errorf("ParsePredicate(%q) = nil error, want error", in)
		}
	}
}

func TestHasOperator(t *testing.T) {
	yes := []string{"a==b", "a!=b", "a>b", "a>=b", "a<b", "a<=b", "a~b"}
	for _, s := range yes {
		if !HasOperator(s) {
			t.Errorf("HasOperator(%q) = false, want true", s)
		}
	}
	no := []string{"file.jsonl", "plainword", "a=b", "a b"}
	for _, s := range no {
		if HasOperator(s) {
			t.Errorf("HasOperator(%q) = true, want false", s)
		}
	}
}

// rec builds a record from key/value pairs; numbers use json.Number to match
// what the reader produces.
func rec(kv ...any) *Record {
	r := NewRecord()
	for i := 0; i+1 < len(kv); i += 2 {
		r.Set(kv[i].(string), kv[i+1])
	}
	return r
}

func TestPredicateMatch(t *testing.T) {
	tests := []struct {
		name string
		pred string
		rec  *Record
		want bool
	}{
		// Numeric path: literal parses as a number, record value is a number.
		{"num eq", "code==200", rec("code", json.Number("200")), true},
		{"num eq no", "code==200", rec("code", json.Number("500")), false},
		{"num ne", "code!=200", rec("code", json.Number("500")), true},
		{"num gt", "code>200", rec("code", json.Number("500")), true},
		{"num gt no", "code>200", rec("code", json.Number("200")), false},
		{"num ge", "code>=200", rec("code", json.Number("200")), true},
		{"num lt", "code<300", rec("code", json.Number("200")), true},
		{"num le", "code<=200", rec("code", json.Number("200")), true},
		{"num float", "latency>=12.5", rec("latency", json.Number("12.5")), true},
		{"num float64 value", "latency<20", rec("latency", 12.5), true}, // float64 also numeric

		// Numeric literal vs non-numeric record value → not comparable → false
		// for every operator, including !=.
		{"num vs string value eq", "code==200", rec("code", "200"), false},
		{"num vs string value ne", "code!=200", rec("code", "200"), false},
		{"num vs null value", "code==200", rec("code", nil), false},

		// String path: literal is not a number.
		{"str eq", "level==info", rec("level", "info"), true},
		{"str eq no", "level==info", rec("level", "error"), false},
		{"str ne", "level!=info", rec("level", "error"), true},
		{"str lt", "level<info", rec("level", "error"), true}, // "error" < "info"
		{"str gt", "level>info", rec("level", "warn"), true},  // "warn" > "info"
		{"bool eq via string", "retry==true", rec("retry", true), true},
		{"bool ne via string", "retry!=false", rec("retry", true), true},

		// Substring (~) is string-only, works on any value's string form.
		{"contains", "msg~oo", rec("msg", "boom"), true},
		{"contains no", "msg~xyz", rec("msg", "boom"), false},
		{"contains number form", "code~0", rec("code", json.Number("200")), true},

		// Missing field never matches, for any operator.
		{"missing eq", "level==info", rec("other", "x"), false},
		{"missing ne", "level!=info", rec("other", "x"), false},
		{"missing contains", "level~in", rec("other", "x"), false},
	}
	for _, tt := range tests {
		p, err := ParsePredicate(tt.pred)
		if err != nil {
			t.Fatalf("%s: ParsePredicate(%q): %v", tt.name, tt.pred, err)
		}
		if got := p.Match(tt.rec); got != tt.want {
			t.Errorf("%s: %q match = %v, want %v", tt.name, tt.pred, got, tt.want)
		}
	}
}

func TestFilterAndsPredicates(t *testing.T) {
	records := []*Record{
		rec("level", "info", "code", json.Number("200")),
		rec("level", "error", "code", json.Number("500")),
		rec("level", "info", "code", json.Number("500")),
	}
	preds := []Predicate{
		mustParse(t, "level==info"),
		mustParse(t, "code>=300"),
	}
	got := Filter(records, preds)
	if len(got) != 1 {
		t.Fatalf("Filter matched %d records, want 1", len(got))
	}
	if v, _ := got[0].Get("code"); v.(json.Number).String() != "500" {
		t.Errorf("matched record code = %v, want 500", v)
	}
}

func TestFilterNoPredicatesReturnsAll(t *testing.T) {
	records := []*Record{rec("a", "1"), rec("a", "2")}
	if got := Filter(records, nil); len(got) != 2 {
		t.Errorf("Filter(_, nil) returned %d, want 2", len(got))
	}
}

// TestFilterExactNumericGiants pins the first class the float64 comparator got
// wrong: integers past float64's 53-bit mantissa. 9007199254740993 and
// 9007199254740992 both round to 9007199254740992.0, so the old float compare
// called them equal. Here both the record value and the predicate literal are
// exact integer literals (each still parses under Q3's unchanged inference), so
// the exact math/big.Rat path decides every operator, in both directions.
func TestFilterExactNumericGiants(t *testing.T) {
	const (
		big1 = "9007199254740993" // 2^53 + 1
		big2 = "9007199254740992" // 2^53
	)
	tests := []struct {
		name      string
		val, pred string // record value and predicate literal
		op        string
		want      bool
	}{
		{"993>992", big1, big2, ">", true},
		{"992>993", big2, big1, ">", false},
		{"993<992", big1, big2, "<", false},
		{"992<993", big2, big1, "<", true},
		{"993==992", big1, big2, "==", false}, // float would say true
		{"993==993", big1, big1, "==", true},
		{"993!=992", big1, big2, "!=", true}, // float would say false
		{"992!=992", big2, big2, "!=", false},
	}
	for _, tt := range tests {
		p := mustParse(t, "n"+tt.op+tt.pred)
		if got := p.Match(rec("n", json.Number(tt.val))); got != tt.want {
			t.Errorf("%s: %q on n=%s = %v, want %v", tt.name, "n"+tt.op+tt.pred, tt.val, got, tt.want)
		}
	}
}

// TestFilterExactNumericOverflow pins the second class: magnitudes past
// float64's range. As record values, 10e400 and 9e400 overflow float64 to +Inf,
// so the old numericValue(Float64) path errored and dropped the record for every
// operator; exact comparison against a finite literal instead orders them
// correctly. Q3 inference is unchanged — an overflow *literal* still fails
// ParseFloat and stays a string predicate — so the literal compared here is
// finite while the giant magnitudes live on the record side (mirroring sort,
// which compares two record values). The direct 10e400-vs-9e400 ordering is
// pinned in TestCompareJSONNumbersExact.
func TestFilterExactNumericOverflow(t *testing.T) {
	const (
		huge1 = "10e400" // larger
		huge2 = "9e400"  // smaller
		ref   = "1e6"    // finite reference, far below both magnitudes
	)
	tests := []struct {
		name string
		val  string // record value (overflows float64)
		op   string
		want bool
	}{
		{"10e400>1e6", huge1, ">", true}, // old float path: false (dropped)
		{"9e400>1e6", huge2, ">", true},  // old float path: false (dropped)
		{"10e400<1e6", huge1, "<", false},
		{"9e400<1e6", huge2, "<", false},
		{"10e400==1e6", huge1, "==", false},
		{"9e400==1e6", huge2, "==", false},
		{"10e400!=1e6", huge1, "!=", true}, // old float path: false (dropped)
		{"9e400!=1e6", huge2, "!=", true},  // old float path: false (dropped)
	}
	for _, tt := range tests {
		pred := "n" + tt.op + ref
		p := mustParse(t, pred)
		if got := p.Match(rec("n", json.Number(tt.val))); got != tt.want {
			t.Errorf("%s: %q on n=%s = %v, want %v", tt.name, pred, tt.val, got, tt.want)
		}
	}
}

func mustParse(t *testing.T, s string) Predicate {
	t.Helper()
	p, err := ParsePredicate(s)
	if err != nil {
		t.Fatalf("ParsePredicate(%q): %v", s, err)
	}
	return p
}
