# logq — requirements (v0.2)

> Revision 2026-07-14 (M2): owner-signed; landed via operator branch PR.
> Changes vs v0: nested-field access (shipped in v0.1 by #15, which lifted the
> v0 exclusion), the v0.1 subcommands (`distinct`, `sort`, `filter --count`),
> and the v0.2 exactness semantics + new capabilities (#21–#25).

**Product:** `logq`, a command-line tool that reads JSON-lines logs and answers
questions about them. Input is one JSON object per line, from stdin or file
arguments. Output goes to stdout.

## Functional requirements

- `logq fields [--sort]` — list the field keys seen across records, each with
  its observed value type(s) and a count of records containing it. Default
  order is first-seen; `--sort` orders deterministically (v0.2, #25 — final
  semantics per the issue's Clarify outcome).
- `logq filter <predicate>...` — print only records matching all given
  predicates (logical AND). A predicate is `field OP value`; operators:
  `==`, `!=`, `>`, `>=`, `<`, `<=`, `~` (substring contains).
  `--count` prints only `{"count": N}` through the standard formatter (v0.1).
  **Exactness (v0.2, #21):** when the record value is a JSON number and the
  predicate literal is numeric, ordering/equality operators compare the exact
  literals (arbitrary precision), never a float64 approximation.
- `logq stats --group-by <field> [--field <numeric-field>]` — group records and
  report per group: record count, and (with `--field`) min/max/sum/avg.
  **Exactness (v0.2, #22):** min/max select by exact comparison and render the
  original JSON literal; sum is exact over all-integer groups; avg remains a
  documented float64 approximation.
- `logq distinct <field> [--top N]` — each distinct value of a field with its
  occurrence count; value-ascending by canonical rendering (v0.1). `--top N`
  (v0.2, #24) limits to the N most frequent (count-desc, value-asc tie-break).
- `logq sort --by <field> [--desc]` — records ordered by a field; JSON-number
  pairs compare exactly (big.Rat over literals, v0.1 #14 review fix); missing
  fields sort last; stable (v0.1).
- `logq head [-n N]` / `logq tail [-n N]` — first/last N records (default 10);
  N ≤ 0 is a usage error (head is v0.2, #23).
- **Nested fields (v0.1, #15):** every field-taking position (`filter`
  predicates, `stats --group-by/--field`, `distinct`, `sort --by`) resolves
  dot-paths `a.b.c` — an exact top-level key wins over traversal; traversal
  through a non-object or absent segment resolves to "missing". `fields`
  remains top-level-only.
- Global `--format table|json|logfmt` (default `table`) honored by every
  subcommand; global input handling: stdin when no file args, else the file
  args concatenated in order.

## Resolved clarifications (carried from v0, still binding)

- **Q2 malformed / non-JSON lines:** skip by default and count skipped;
  `--strict` makes any malformed line a non-zero-exit error.
- **Q3 `filter` value typing:** infer from the literal — a number literal
  compares numerically (exactly, from v0.2), otherwise string; `~` is
  string-only; missing field means the predicate is false. Type-mismatch is
  non-matching for every operator including `!=` (v0 worker judgment call,
  owner-reviewed).
- **Q4 `stats`:** count always; aggregates only with `--field`; single
  `--group-by` key; non-numeric values under `--field` are skipped and counted.
- **Q5/Q7:** no `--follow`, no time semantics (unchanged; still out of scope).
- **Numbers are literals:** the reader preserves numeric literals verbatim
  (`json.Number`); rendering never round-trips them through float64.

## Non-functional

- Deterministic, hermetic: no network, no clock/timezone dependence, no
  external services. Same input → same output.
- Single self-contained Go binary; stdlib only; `go build ./...` and
  `go test ./...` are the only toolchain needs.
- Reasonable memory on large inputs is a nice-to-have, not a requirement.

## Explicitly out of scope (v0.2)

- No `tail --follow` / streaming / real-time mode.
- No time parsing, time-window filters, or sort-by-time.
- No config files, no plugins, no output to files (stdout only).
- No array indexing or wildcards in dot-paths; `fields` stays top-level-only.
- No exact `avg` (documented float64 approximation); no new aggregates.
- No multi-key sort.
