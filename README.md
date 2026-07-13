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
