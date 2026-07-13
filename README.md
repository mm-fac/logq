# logq

`logq` is a small, hermetic command-line tool that reads JSON-lines logs (one
JSON object per line) and answers questions about them. Input comes from stdin
or from file arguments (concatenated in order); output goes to stdout. It has no
dependencies beyond the Go standard library.

See [`requirements.md`](requirements.md) for the v0 specification.

## Build

```
go build -o logq ./cmd/logq
```

## Global usage

```
usage: logq [--format table|json|logfmt] [--strict] <command> [flags] [files...]
```

- `--format table|json|logfmt` — output rendering, honored by every subcommand.
  Default is `table`. May appear before or after the subcommand.
- `--strict` — treat any malformed (non-JSON) input line as a fatal error and
  exit non-zero. Without it, malformed lines are skipped and a count is written
  to stderr while processing continues.
- Input source: stdin when no file arguments are given, otherwise the named
  files concatenated in order.

Exit codes: `0` success, `1` runtime error (e.g. missing file, strict malformed
line), `2` usage error (unknown command, bad flag, bad predicate).

All examples below run against the committed fixture
[`testdata/events.jsonl`](testdata/events.jsonl):

```
{"ts":"2026-07-13T00:00:01Z","level":"info","method":"GET","path":"/","status":200,"ms":10}
{"ts":"2026-07-13T00:00:02Z","level":"info","method":"GET","path":"/about","status":200,"ms":20}
{"ts":"2026-07-13T00:00:03Z","level":"warn","method":"POST","path":"/login","status":401,"ms":50}
{"ts":"2026-07-13T00:00:04Z","level":"error","method":"GET","path":"/api","status":500,"ms":100}
{"ts":"2026-07-13T00:00:05Z","level":"info","method":"GET","path":"/api","status":200,"ms":30}
{"ts":"2026-07-13T00:00:06Z","level":"error","method":"POST","path":"/api","status":503,"ms":200}
```

## Subcommands

### `fields`

List every field key seen across the records, with its observed value type(s)
and the number of records that contained it.

```
$ ./logq fields testdata/events.jsonl
field   types   count
ts      string  6
level   string  6
method  string  6
path    string  6
status  number  6
ms      number  6
```

### `filter`

Print only the records matching all of the given `field OP value` predicates
(logical AND). Operators: `==` `!=` `>` `>=` `<` `<=` and `~` (substring
contains). A numeric literal compares numerically, otherwise the comparison is
on the value's string form; a record missing the field never matches. Quote
predicates so the shell does not interpret `>`/`<`.

```
$ ./logq filter 'status>=500' testdata/events.jsonl
ts                    level  method  path  status  ms
2026-07-13T00:00:04Z  error  GET     /api  500     100
2026-07-13T00:00:06Z  error  POST    /api  503     200
```

With `--count`, `filter` suppresses the records and prints only the number of
matches as a single `{"count": N}` row, rendered through `--format` like any
other output:

```
$ ./logq filter --count 'status>=500' testdata/events.jsonl
count
2
```

### `stats`

Group records by `--group-by <field>` and report the per-group record count.
With `--field <numeric-field>`, also report min/max/sum/avg of that field;
records whose value is missing or non-numeric are counted under `skipped`.

```
$ ./logq stats --group-by level --field ms testdata/events.jsonl
level  count  min  max  sum  avg  skipped
error  2      100  200  300  150  0
info   3      10   30   60   20   0
warn   1      50   50   50   50   0
```

### `tail`

Print the last N records in input order (default 10; `-n N` to change).

```
$ ./logq tail -n 2 testdata/events.jsonl
ts                    level  method  path  status  ms
2026-07-13T00:00:05Z  info   GET     /api  200     30
2026-07-13T00:00:06Z  error  POST    /api  503     200
```

### `distinct`

List each distinct value of a top-level `<field>` with the number of records
in which it occurred, one row per value. Values of different JSON types are
distinct even when they render alike (the number `1` and the string `"1"`).
Rows are sorted by each value's canonical JSON rendering, ascending. Records
missing the field are skipped and their count is reported on stderr.

```
$ ./logq distinct level testdata/events.jsonl
value  count
error  2
info   3
warn   1
```

### `sort`

Print all records ordered by a top-level `--by <field>`; add `--desc` to reverse
the order. Two JSON-number values compare numerically; any other pair compares by
canonical JSON rendering, bytewise. Records missing the field sort last (keeping
their input order) regardless of `--desc`, and the sort is stable so ties keep
input order.

```
$ ./logq sort --by status testdata/events.jsonl
ts                    level  method  path    status  ms
2026-07-13T00:00:01Z  info   GET     /       200     10
2026-07-13T00:00:02Z  info   GET     /about  200     20
2026-07-13T00:00:05Z  info   GET     /api    200     30
2026-07-13T00:00:03Z  warn   POST    /login  401     50
2026-07-13T00:00:04Z  error  GET     /api    500     100
2026-07-13T00:00:06Z  error  POST    /api    503     200
```

## Nested fields

Every field-taking position — `filter` predicates, `stats --group-by`/`--field`,
`distinct <field>`, and `sort --by` — accepts a dotted path like `user.role` to
reach into nested JSON objects. Resolution follows two rules:

- **Exact key first.** A field argument is first matched against an EXACT
  top-level key, dots and all. Only when no such key exists is the argument split
  on `.` and walked segment by segment (the first segment is a top-level key,
  each later segment indexes the object reached so far). So a record that
  literally has a top-level `"user.role"` key is read from that key, not from a
  nested `user` → `role`.
- **Missing on any dead end.** If an intermediate value is not an object, or a
  segment is absent, the path resolves to "missing" — exactly as an absent
  top-level field does (predicate false / record skipped / sorted last).

`fields` remains top-level-only: it lists the outermost keys and does not expand
nested objects.

The example below runs against [`testdata/nested.jsonl`](testdata/nested.jsonl),
whose records nest `user` and `req` objects:

```
$ ./logq stats --group-by user.role --field req.ms testdata/nested.jsonl
user.role  count  min  max  sum  avg  skipped
admin      2      10   50   60   30   0
guest      2      20   30   50   25   0
```

## Output formats

Every subcommand honors `--format`. For example, the same `filter` query as
JSON-lines (which can be piped back into `logq`) or logfmt:

```
$ ./logq --format json filter 'status>=500' testdata/events.jsonl
{"ts":"2026-07-13T00:00:04Z","level":"error","method":"GET","path":"/api","status":500,"ms":100}
{"ts":"2026-07-13T00:00:06Z","level":"error","method":"POST","path":"/api","status":503,"ms":200}

$ ./logq --format logfmt filter 'status>=500' testdata/events.jsonl
ts=2026-07-13T00:00:04Z level=error method=GET path=/api status=500 ms=100
ts=2026-07-13T00:00:06Z level=error method=POST path=/api status=503 ms=200
```

## Reading from stdin

With no file arguments, `logq` reads stdin. The result is identical to passing
the same data as a file:

```
$ cat testdata/events.jsonl | ./logq stats --group-by level
level  count
error  2
info   3
warn   1
```

## Determinism

`logq` is deterministic and hermetic: the same input always produces the same
output (no clock, timezone, network, or filesystem dependence beyond reading the
named inputs). This is enforced in CI by `TestDeterministicOutput`
(`cmd/logq/determinism_test.go`), which runs every subcommand in every format
multiple times on a committed fixture and asserts byte-identical output. CI runs
`go build ./...` and `go test ./...`.

## Development

```
go build ./...
go vet ./...
go test ./...
```
