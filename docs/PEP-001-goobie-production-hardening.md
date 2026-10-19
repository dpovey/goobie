# PEP-001: Goobie Production Hardening

**Status:** Draft  
**Date:** 2026-09-07  
**Author:** Dean Povey  

## Context

Goobie ports Doobie's core idea — SQL as composable values, programs as `Op[A]`, typed query terminals — but several production gaps remain: no `context.Context`, fragile placeholder rewriting, `lib/pq`, incomplete batches, and no column codecs. The Go ecosystem already covers static SQL (`sqlc`) and ORMs; goobie's niche is **dynamic, composable SQL programs** inside transactions.

This PEP sequences the highest-leverage work from the Doobie comparison, including a Doobie-like interpolator that **expands to existing `Fragment` constructors** (`Fr`, `FrConst`, `Combine`).

### Doobie `sql"..."` vs Go

Doobie:

```scala
sql"SELECT name FROM person WHERE age > ${p.age + 1} AND country = $country"
sql"SELECT * FROM person WHERE $where AND name = $name"  // $where splices a Fragment
```

`$ident` binds a simple name; `${expr}` is a Scala expression. Go has no interpolators, so this syntax is a **constant string** plus **codegen** that lifts each hole into the API we already have.

Authoring (raw string so `${` is literal Go):

```go
fr := goobie.SQL(`SELECT name FROM person WHERE age > ${p.Age + 1} AND country = $country`)
where := goobie.SQL(`age > ${min}`)
q := goobie.SQL(`SELECT * FROM person WHERE ${where} AND name = ${p.Name}`)
```

Generate / `go/analysis -fix` rewrites to fragments:

```go
fr := goobie.Fr("SELECT name FROM person WHERE age > ? AND country = ?", p.Age+1, country)
q := goobie.Combine([]goobie.Fragment{
    goobie.FrConst("SELECT * FROM person WHERE "),
    where,
    goobie.Fr(" AND name = ?", p.Name),
})
```

All binds in one stretch of SQL become one `Fr` with `?` in order (same as writing fragments by hand). A `Fragment`-typed hole splits the string and `Combine`s. `ToPostgres` stays the dialect step — interpolator does not own `$n`.

Do **not** emit sqlf. sqlf’s `%s` / nested `*Query` is a parallel fragment model; goobie already has `Fr` + `Args` + splice via `Combine`. A second representation adds a dependency and two ways to write the same SQL.

#### Grammar (deliberately dumb)

| Token | Meaning |
|-------|---------|
| `$${` | literal `${` in SQL |
| `$$` | literal `$` (Doobie parity) |
| `${` … **first `}`** | Go expression — **no `{` or `}` in the body** |
| `$ident` | `[A-Za-z_][A-Za-z0-9_]*` (Doobie `$name`) |
| `$` otherwise | generate error |

No brace matching, no nested interpolations. Then `go/parser.ParseExpr` on the span; `go/packages` types: `Fragment` → splice, else bind.

Valid: `${p.Age+1}`, `${foo(bar)}`, `${xs[i]}`, `${pkg.Max(a, b)}`, `${where}`.

Invalid — hoist to a local: `${Person{Name: n}}`, `${[]int{1,2}}`, `${func() int { return 1 }}`.

#### Edge cases (document; do not add a real Go lexer in v1)

1. **`}` inside the expression** (`${"}"}`, comments `/* } */`): first `}` wins; parse fails. Assign to a variable.
2. **Composite / map / func literals** need `{`. Same workaround.
3. **Interpolator runs on the Go string**, like Scala — including inside SQL quotes. Write `= ${name}`, not `= '${name}'`. Literal `${` is `$${`.
4. **Handwritten `$1`** is not `$ident`; error. Don’t mix Postgres placeholders with `$` interpolations.
5. **Generics** `${foo[T](x)}` are fine (no braces).

Runtime `Fr(sql, args...)` stays as the low-level API. Map-style `${name}` is rejected. querygen / sqlc stay optional and orthogonal.

## Waves

### Wave 1: Go citizenship

Make goobie safe to run in production Go services before adding features.

**Tasks:**

- [ ] Add `context.Context` to `Runner`, `Transactor`, and `Transaction`:
  - `Query(ctx, Fragment) (Rows, error)`
  - `RunUpdate(ctx, UpdateT) (int64, error)`
  - `RunUpdateMany(ctx, UpdateT, [][]any) (int64, error)`
  - `Transact(ctx, func(Transaction) error) error`
  - `BeginTx(ctx) (Transaction, error)`
- [ ] Thread context through `Op` runners (`Run(ctx, Runner)` or context stored on `Runner` — pick one; prefer explicit `ctx` on `Run` and `Transact`).
- [ ] Fix `Transact` commit error handling in `pkg/goobie/sql/transactor.go` (named return so deferred commit error is not dropped).
- [ ] Add pgx adapter (`pkg/goobie/pgx` or extend `pkg/goobie/sql`):
  - `pgxpool.Pool` / `pgx.Tx` implementing `Transactor` / `Transaction`
  - Keep `database/sql` adapter for compatibility; document pgx as recommended for PostgreSQL
- [ ] Stream lifetime: document that `QueryT.Stream` must not outlive the transaction/connection that opened `Rows`; add test proving cursor closes when `Op` completes inside `Transact`
- [ ] Delete stale `cmd/generator` (regex `${}` without types; `doobie.fragment`). Replaced by Wave 2 interpolator.

**Gate:**

- [ ] `go vet ./...` passes
- [ ] `go test -race ./...` passes (Postgres via docker-compose)
- [ ] Context cancellation aborts an in-flight query (new test)
- [ ] Failed commit surfaces as error from `Transact` (new test)

**Files (expected):**

| File | Change |
|------|--------|
| `pkg/goobie/transactor.go` | Context on interfaces |
| `pkg/goobie/op.go` | Context through `Run` |
| `pkg/goobie/sql/*.go` | Context + transact fix |
| `pkg/goobie/pgx/*.go` | New pgx adapter |
| `pkg/goobie/query.go` | Pass ctx to `Runner.Query` |
| `internal/example/*` | Update call sites |

---

### Wave 2: Trustworthy fragments + Doobie interpolator

Harden `?` → `$n`. Add `${expr}` / `$ident` codegen that emits `Fr` / `FrConst` / `Combine`.

**Tasks — fragment compiler:**

- [ ] Fix `ToPostgres` / placeholder rewrite (quotes, PG JSON `?`, `[]byte`, batch SQL with no args)
- [ ] `In` / `NotIn`, `Where`, `Join` on `Fragment`
- [ ] `Validate()` placeholder count vs `len(Args)`

**Tasks — interpolator:**

- [ ] Analyzer / `go generate` expands `goobie.SQL(\`...\`)` constant strings per grammar above
- [ ] Consecutive binds → one `Fr("... ? ...", exprs...)`
- [ ] `Fragment`-typed hole → `Combine` of `FrConst` / `Fr` / the hole
- [ ] `ParseExpr` + type-check; generate error on `{`/`}` in `${}` body, on `$1`, on unclosed `${`
- [ ] Golden tests: bind, splice, `$ident`, `$$` / `$${`, invalid composite literal
- [ ] Delete `cmd/generator` (regex-only, `doobie.fragment`)

**Gate:**

- [ ] `go test -race ./...` passes
- [ ] Interpolator goldens pass
- [ ] Example uses `${expr}` + splice; generate then test
- [ ] No sqlf dependency

**Non-goals:** sqlf; querygen; `.sql` sqlc generate; runtime map interpolator; brace-matching expressions.

---

### Wave 3: Column codecs (Get / Put)

Doobie's typeclass layer for DB types. In Go: a small registry, not a typeclass hierarchy.

**Tasks:**

- [ ] Define `read.ColumnReader[T]` / `write.ColumnWriter[T]` (or combined `Codec[T]`) with registration for:
  - `uuid.UUID` (github.com/google/uuid)
  - `json.RawMessage` / `[]byte`
  - `time.Time` (confirm `database/sql` defaults)
  - `pgtype` mappings when using pgx adapter
- [ ] Integrate with `read.Get[T]()` via registry lookup before fallbacks
- [ ] Integrate with `write.Writer[T]()` symmetrically
- [ ] Document extension point for enums (`Scan`/`Value` wrappers)

**Gate:**

- [ ] Round-trip tests: insert UUID/JSON, read back via `Query0` / `Query`
- [ ] `go test -race ./pkg/goobie/read/... ./pkg/goobie/write/...`

---

### Wave 4: Batches and RETURNING

Close the gap with Doobie `updateMany` / `withUniqueGeneratedKeys`.

**Tasks:**

- [ ] Fix `UpdateWithWrite.GetArgs()` to support all `Values`, not only `[0]`
- [ ] Typed `UpdateMany[A]` using `Writer[A]` → `[][]any` internally
- [ ] Add `Returning` fragment helper or `UpdateReturning[A, R]` terminal
- [ ] `QueryReturning` / `UniqueReturning` for `INSERT ... RETURNING id`
- [ ] Replace `lastval()` example pattern with RETURNING where appropriate

**Gate:**

- [ ] Batch insert 3+ rows, assert row count
- [ ] `INSERT ... RETURNING` → `Unique[int64]` test
- [ ] `UpdateAll` partial count on mid-batch error (existing behavior preserved)

---

### Wave 5: Named struct mapping (optional)

Positional struct scan surprises sqlx users. Keep `RowN` for explicit column order.

**Tasks:**

- [ ] Add `db` tag support to `product.Productizer` (column name → field)
- [ ] `QueryNamed[A]` or opt-in on `Query[A]` via struct tag presence
- [ ] Document: positional default, named opt-in; `RowN` remains the zero-reflection path

**Gate:**

- [ ] Test column order mismatch with `db` tags still scans correctly
- [ ] Benchmark: confirm `RowN` path unchanged

---

## Security Considerations

### Threat Model

SQL injection via string concatenation; credential exposure in logs; transaction boundary errors causing partial commits.

### Mitigations

| Risk | Mitigation |
|------|------------|
| SQL injection | Interpolator emits `Fr` binds / `Combine` splices; never concatenates expr values into SQL; `In` never concatenates values |
| Secrets in logs | No SQL/args logging in core (defer to caller); document safe logging |
| Partial commit | Fix `Transact` commit error propagation; fail-closed rollback on error |

### Not Addressed

Connection pool tuning, RLS, audit logging, query allowlists.

## Consequences

1. **Benefits**
   - Goobie becomes viable alongside sqlc (static) rather than competing with it
   - Compositional SQL matches Doobie's main differentiator
   - pgx + context align with production Go expectations

2. **Trade-offs**
   - Context threading is a breaking API change
   - `${expr}` codegen is goobie-owned (no library); expressions cannot contain `{`/`}`
   - Named mapping adds reflection cost; `RowN` stays the performance path

3. **Future work (deferred)**
   - `MapN` without `Op[any]` (Go invariance limits)
   - Stream inside `Op` with fs2-like resource safety (evaluate after Wave 1 stream tests)
   - Multi-dialect placeholder backends beyond PostgreSQL
   - README and migration guide from `lib/pq` to pgx
   - Optional sqlc-style `.sql` → `QueryT` generate

## Execution order summary

| Wave | Deliverable | Blocks |
|------|-------------|--------|
| 1 | context, transact fix, pgx | everything else |
| 2 | fragment rewrite, `In`/`Join`, `${expr}` → `Fr`/`Combine` | dynamic SQL trust |
| 3 | codecs | ergonomic typed columns |
| 4 | batches, RETURNING | insert-heavy apps |
| 5 | named scan | sqlx refugees |

**Start Wave 1 immediately.** Wave 2: fix fragments, then interpolator → `Fr`/`Combine`.

**Updated:** 2026-09-07 (Doobie `${expr}` expands to existing fragments, not sqlf)
