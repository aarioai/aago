---
name: aago-mysqlx-queries
description: >-
  Write MySQL 8.0+ queries with the aago driver mysqlx package: DB/Tx execution with *ae.Error results,
  Cond and `asql.FilterExpr` condition building, struct driven WHERE clauses, safe increments, pagination and
  sharded-table UNION helpers. Use when adding or refactoring SQL access code, or when a query
  builder produces the wrong SQL fragment.
---

# Writing queries with mysqlx

## When to use this skill

- Implementing or refactoring repository code that talks to MySQL 8.0+ through this repository.
- A `Cond`/`asql.FilterExpr`/`Where` result does not match the expected SQL.
- Deciding between `DB`, `Tx` and `ORMS`, or between `Query` and `Scan*`.

## Ground rules

1. **Values are placeholders.** Pass untrusted values as `?` arguments. `asql.FilterExpr`/`defenseInjection`
   are a safety net for filtered input, not a substitute.
   `Cond.Write`/`WriteString` are for trusted SQL fragments only.
2. **`*ae.Error` everywhere.** `mysqlx` never returns `error`; check `e.IsNotFound()` for "no row".
3. **Close what you open.** `Query` returns `*sql.Rows` and `Prepare` returns `*sql.Stmt`; the
   caller must close them.
4. **Transactions need `defer tx.Recover()()`** — two calls; a forgotten commit is rolled back.
5. **`ORMS` needs an entity** with `Table()` and `Indexes()` (single-column primary key).

## Reading and writing

```go
sdb := mysqlx.NewDriver(schema, db, nil) // third arg: deferred error, keep nil normally

// single row
var name string
e := sdb.ScanAny(ctx, "SELECT `name` FROM `user` WHERE `id`=?", 100, &name)

// insert / update
id, e := sdb.Insert(ctx, "INSERT INTO `user` (`name`) VALUES (?)", "Aario")
n, e := sdb.Update(ctx, "UPDATE `user` SET `name`=? WHERE `id`=?", "Aario", id)

// many rows
rows, e := sdb.Query(ctx, "SELECT `id`,`name` FROM `user` WHERE `status`=?", 0)
if e == nil {
    defer rows.Close()
    for rows.Next() { /* rows.Scan(...) */ }
}
```

`Scan`, `ScanX`, `ScanAny` and `ScanArgs` differ only in how the single query argument is passed.

## Conditions

```go
cond := mysqlx.NewCond(paging)           // atype.Paging
cond.And("t.user_id", "100")           // asql.FilterExpr grammar
cond.Write("AND", aenum.StsInvalid("t.status"))
cond.Try("t.id DESC", 0, 20)

suffix, args := cond.StmtArgs()        // preferred: ? placeholders + args
qs := "SELECT * FROM `tb` t" + suffix
if e := cond.Err(); e != nil {
    return e                           // Stmt/StmtArgs fail closed with WHERE 1!=1
}
```

`asql.FilterExpr` grammar (`asql.NewFilterExpr`; `FilterExpr` renders placeholders):

| Input | SQL |
| --- | --- |
| `Aario` / `:Aario` | equality |
| `:@Aario` / `:Aario@` / `:@Aario@` | ends with / starts with / contains |
| `:Aario,Tom` | `IN ('Aario','Tom')` |
| `:start~end`, `:start~`, `:~end` | range / lower bound / upper bound |

`@`, `~` and `,` are reserved and cannot be escaped. Legacy `::Aario` / `:Aario:` is rejected;
invalid grammar sets `Err()` and renders `1!=1` instead of dropping the filter. LIKE wildcards
(`%`, `_`, `!`) are escaped with `ESCAPE '!'`, field names are validated and back-quoted.

Struct helpers: `mysqlx.And(u, "name", "age")` (column names) or `mysqlx.And(u, "Name", "Age")`
(Go fields), plus `Or`, `Like`, `Comma`, `AndWithWhere`, `OrWithWhere`, `Where`. The literal forms
escape values (`'` → `''`, `\` → `\\`) and fail closed with `1!=1`; `AndArgs` / `OrArgs` /
`LikeArgs` return `?` placeholders plus arguments and are preferred for untrusted data.

All builders live in `../../../../aago/driver/asql` and only take a `Dialect`; `mysqlx` and `postgresx` are thin
wrappers over them. Add new statement logic to `asql`, not to a driver.

## Bulk and sharded helpers

```go
qs, e := mysqlx.InsertMany("user", "(a,b)", "(?,?)", len(items))
if e != nil { return e }
where := mysqlx.In("id", idSet)                              // "`id` IN (…)" / "1!=1"
rows, e := mysqlx.InUnionAllQs(ctx, sdb, format, ids, func(id uint64) []any { ... })
```

`mysqlx.InArgs` (`"(...)"`) marks the generated `IN` list inside union query formats; the helpers
deduplicate identical branches, inline `LIMIT n` and return `ae.ErrorEmptyInput` for an empty id
list.

## Pagination and limits

`Cond.LimitN()` defaults to 10 rows; `Cond.Try` only fills values that are still empty, so
caller-supplied pagination wins.

## Verification

- Unit-test pure builders (`asql.FilterExpr`, `Cond.Stmt`, `Where`, `InsertMany`) instead of hitting a
  database.
- `go test ./driver/mysqlx/...` and `go vet ./...` must pass.
- Full reference: `../../../../aago/doc/driver/mysqlx.md`.
