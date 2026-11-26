# postgresx — PostgreSQL helpers

Package: `github.com/aarioai/aago/driver/postgresx`

`postgresx` uses native `pgx/v5` and `pgxpool`. Create and configure the pool with pgx, pass it
to `NewDriver`, and close the pool when the application shuts down:

```go
pool, err := pgxpool.New(ctx, dsn)
if err != nil {
    return err
}
defer pool.Close()

db := postgresx.NewDriver("app", pool, nil)
id, e := db.Insert(ctx,
    "INSERT INTO users (name) VALUES ($1) RETURNING id",
    "Aario",
)
if e != nil {
    return e
}
```

`Insert` requires a PostgreSQL `RETURNING` clause. `Update` returns the number of affected rows.
`Query` returns `pgx.Rows`; callers must close it.

## Transactions

```go
tx, e := db.Begin(ctx, pgx.TxOptions{})
if e != nil {
    return e
}
defer tx.Recover(ctx)()

if e := tx.Exec(ctx, "UPDATE users SET name=$1 WHERE id=$2", "Aario", id); e != nil {
    return e
}
return tx.Commit(ctx)
```

An uncommitted transaction is rolled back by `Recover`; a panic is rethrown after rollback.
`Prepare` pins one pooled connection until its `PreparedStmt.Close()` is called. Close outstanding
rows before releasing the statement.

## Conditions

`../../aago/driver/asql` contains the shared dialect layer: filter grammar, identifier validation, the
`Cond` builder, struct-field helpers, INSERT/IN/UNION builders and the clamped increment
expressions. Both mysqlx and postgresx only supply their `Dialect`, so each driver renders its own
bind markers:

```go
cond := postgresx.NewCond(paging)
cond.And("u.id", "100").Or("u.name", ":@Aario@")
cond.OrderBy("u.created_at DESC")
query, args := cond.StmtArgs()
// query uses PostgreSQL markers: WHERE "u"."id"=$1 OR "u"."name" LIKE $2 ...
rows, e := db.Query(ctx, "SELECT * FROM users u"+query, args...)
```

Use `Cond.StmtArgs()` and bind parameters for query execution. `Write` and `WriteString` accept
trusted SQL fragments only. Invalid grammar, identifiers, or sort expressions fail closed.

PostgreSQL-native helpers include `InsertMany` (`$1`, `$2`, ...), `In` / `InValues`, `ArgPairs`,
struct-driven conditions, `ORMS`, and `UNION ALL` builders. All of them are thin wrappers around
[`../../aago/driver/asql`](./asql.md), which owns the shared statement logic and only needs the PostgreSQL
dialect. Union `format` strings and callback results are trusted SQL fragments; values must be
bound rather than interpolated.

## Verification

```text
go test ./driver/asql/... ./driver/mysqlx/... ./driver/postgresx/...
go vet ./driver/asql/... ./driver/mysqlx/... ./driver/postgresx/...
```
