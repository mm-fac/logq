# logq — requirements (v0)

**Product:** `logq`, a command-line tool that reads JSON-lines logs and answers
questions about them. Input is one JSON object per line, from stdin or file
arguments. Output goes to stdout.

## Functional requirements

- `logq fields` — list the field keys seen across records, each with its
  observed value type(s) and a count of records containing it.
- `logq filter <predicate>...` — print only records matching all given
  predicates (logical AND). A predicate is `field OP value`; operators:
  `==`, `!=`, `>`, `>=`, `<`, `<=`, `~` (substring contains).
- `logq stats --group-by <field> [--field <numeric-field>]` — group records by
  a field and report per group: record count, and (when `--field` is given)
  min/max/sum/avg of that numeric field.
- `logq tail [-n N]` — print the last N records (default 10).
- Global `--format table|json|logfmt` (default `table`) honored by every
  subcommand.
- Global input handling: read from stdin when no file args, else concatenate
  the file args in order.

## Resolved clarifications (Clarify seat, 2026-07-13 — verdict GO)

- **Q1 input source:** both — stdin when no file args, else the file args
  concatenated in order.
- **Q2 malformed / non-JSON lines:** skip by default and count skipped;
  `--strict` makes any malformed line a non-zero-exit error.
- **Q3 `filter` value typing:** infer from the literal — a number literal
  compares numerically, otherwise string; `~` is string-only; missing field
  means the predicate is false.
- **Q4 `stats`:** count always; min/max/sum/avg only with `--field`; single
  `--group-by` key in v0; non-numeric values under `--field` are skipped and
  counted as skipped.
- **Q5 `tail --follow`:** out of scope in v0 (protects hermetic tests).
- **Q6 output formats:** default `table`; all subcommands honor all three
  formats via the shared formatter.
- **Q7 time semantics:** none in v0 — records are opaque key/value objects;
  no sort/filter by time.

## Non-functional

- Deterministic, hermetic: no network, no clock/timezone dependence, no
  external services. Same input → same output.
- Single self-contained Go binary; `go build ./...` and `go test ./...` are
  the only toolchain needs.
- Reasonable memory on large inputs is a nice-to-have, not a v0 requirement.

## Explicitly out of scope (v0)

- No `tail --follow` / streaming / real-time mode.
- No time parsing, time-window filters, or sort-by-time.
- No config files, no plugins, no output to files (stdout only).
- No nested-field access (e.g. `a.b.c`); v0 operates on top-level keys only.
