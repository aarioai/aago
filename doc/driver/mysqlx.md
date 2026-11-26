# mysqlx — MySQL 8.0+ helpers

Package: `github.com/aarioai/aago/driver/mysqlx`

`mysqlx` wraps `database/sql` with `*ae.Error` results, condition builders and a tiny O-R-M.

It targets **MySQL 8.0+** only: there are no MySQL 5.7 fallbacks. Statement building is shared
with `postgresx` through [`../../aago/driver/asql`](./asql.md); `mysqlx` only supplies the MySQL dialect
(back-quoted identifiers, `?` markers, `LIMIT offset,limit`).

## DB

```go
schema, db, e := driver.NewMysqlPool(app, "test")
if e != nil {
    return e
}
sdb := mysqlx.NewDriver(schema, db, nil)   // nil: no deferred error
```

| Method                                        | Result       | Notes                       |
|-----------------------------------------------|--------------|-----------------------------|
| `Prepare(ctx, query)`                         | `*sql.Stmt`  | caller closes the statement |
| `Execute(ctx, query, args...)`                | `sql.Result` |                             |
| `Exec(ctx, query, args...)`                   | `*ae.Error`  | discards the result         |
| `Insert(ctx, query, args...)`                 | `uint`       | last insert id              |
| `Update(ctx, query, args...)`                 | `int64`      | rows affected               |
| `QueryRow(ctx, query, args...)`               | `*sql.Row`   |                             |
| `Query(ctx, query, args...)`                  | `*sql.Rows`  | caller closes the rows      |
| `ScanRow / ScanArgs / Scan / ScanX / ScanAny` | `*ae.Error`  | scan into destinations      |
| `Begin(ctx, opts)`                            | `*Tx`        |                             |

`NewDriver` accepts a deferred error; when it is non-nil every method returns it immediately,
which lets callers build a `*DB` inside a chain without nil checks.

> 中文提示：`NewDriver(schema, db, e)` 的第三个参数用于“延迟错误”，
> 传入非空错误后所有方法都会直接返回该错误，方便链式调用。

## Tx

```go
tx, e := sdb.Begin(ctx, nil)
if e != nil {
    return e
}
defer tx.Recover()()      // rollback on panic or on a forgotten commit

id, e := tx.Insert(ctx, "INSERT INTO user (name) VALUES (?)", "Aario")
if e != nil {
    _ = tx.Rollback()
    return e
}
return tx.Commit()
```

`Recover` must be **called** when deferred: `defer tx.Recover()()`. If the transaction was
neither committed nor rolled back, it is rolled back (earlier versions committed in that case).

> 中文提示：`defer tx.Recover()()` 注意两层括号；未提交的事务会被回滚，而不再被自动提交。

## Conditions

### `Cond` — query builder

```go
cond := mysqlx.NewCond(paging)          // or &mysqlx.Cond{}
cond.And("t.user_id", "100")          // asql.FilterExpr grammar, see below
cond.Write("AND", aenum.StsInvalid("t.status"))
cond.Try("t.id DESC", 0, 20)          // only when not set yet

suffix, args := cond.StmtArgs()       // preferred: WHERE `t`.`user_id`=? … LIMIT 0,20
qs := "SELECT * FROM tb t" + suffix

// or the literal form (escaped, deprecated for untrusted input):
qs = "SELECT * FROM tb t" + cond.Stmt()
//  … WHERE `t`.`user_id`='100' AND t.status<0 ORDER BY t.id DESC LIMIT 0,20
```

| Method                                       | Purpose                                                              |
|----------------------------------------------|----------------------------------------------------------------------|
| `Write(operator, s)`                         | append a raw fragment (operator omitted when the condition is empty) |
| `Args(args…)`                                | append values for placeholders written by `Write`                     |
| `And(field, grammar)` / `Or(field, grammar)` | parse and append an `asql.FilterExpr` condition                       |
| `Concat(operator, field, grammar)`           | `And`/`Or` with an explicit connector                                |
| `OrderBy(keyword)`                           | append to the `ORDER BY` list                                        |
| `Limit(offset, limit)`                       | force the pagination                                                 |
| `Try(orderBy, offset, limit)`                | set pagination only when empty                                       |
| `Stmt()`                                     | literal `WHERE … ORDER BY … LIMIT …` suffix                          |
| `StmtArgs()`                                 | `?` placeholder suffix plus its arguments                            |
| `Err()`                                      | first builder error; `Stmt`/`StmtArgs` then fail closed (`1!=1`)      |

`LimitN()` returns 10 when no limit was set. Only `AND` / `OR` are accepted as connectors.
`Write` and `WriteString` append trusted SQL fragments; never pass user input directly. Bind
untrusted values with `?` placeholders and `Args`.

### `asql.FilterExpr` — compact query grammar

`asql.NewFilterExpr(v)` parses user input in a single-token grammar; an empty value means "no condition":

| Input                             | Meaning                     |
|-----------------------------------|-----------------------------|
| `Aario`                           | `= 'Aario'`                 |
| `:Aario`                          | `= 'Aario'` (same as above) |
| `:@Aario`                         | ends with `Aario`           |
| `:Aario@`                         | starts with `Aario`         |
| `:@Aario@`                        | contains `Aario`            |
| `:Aario,Tom`                      | `IN ('Aario','Tom')`        |
| `:2019-06-01 00:00:00~2019-06-02` | `>=` and `<`                |
| `:2019-06-01~`                    | `>=` only                   |
| `:~2019-06-01`                    | `<` only                    |

- `@`, `~` and `,` are reserved inside the grammar; there is **no** escape sequence.
- The former `::Aario` / `:Aario:` forms are **rejected** instead of degrading to an equality match.
- Invalid input is reported by `Cond.Err()`; `Cond.Stmt` and `Cond.StmtArgs` render `1!=1` (fail closed).
- Values are capped at 4096 bytes and `IN` lists at 512 elements.

```go
expr, e := asql.NewFilterExpr(":Aario,Tom")
sql, args, e := mysqlx.FilterExpr("t.name", expr)
// sql  = "`t`.`name` IN (?,?)"
// args = []any{"Aario", "Tom"}

cond := mysqlx.NewCond(paging)
cond.And("t.name", ":@Aario@")
cond.Stmt() // WHERE `t`.`name` LIKE '%Aario%' ESCAPE '!' LIMIT ...
```

`FilterExpr` returns placeholders and arguments. `Cond.StmtArgs` is preferred for query execution.
`Cond.Stmt` renders escaped literals; it validates and back-quotes the field name, escapes literals
(`'` → `''`, `\` → `\\`) and escapes LIKE wildcards (`%`, `_`, `!` with `ESCAPE '!'`).

> 中文提示：`asql.FilterExpr` 语法直接取自 URL 查询参数；请优先使用 `Cond.StmtArgs` 走占位符。
> 字段名会做白名单校验；非法语法会让 `Cond.Stmt` 失败关闭（`WHERE 1!=1`）而不是静默丢弃条件。

### `constraint.go` — struct driven conditions

```go
mysqlx.And(user, "name", "age")            // field names are db columns
mysqlx.And(user, "Name", "Age")            // exported Go field names
mysqlx.Like(user, "name")                  // LIKE '%…%' ESCAPE '!'
mysqlx.AndWithWhere(user, "name")          // " WHERE `name`='…' "
mysqlx.Where("`a`=1", "AND")               // trailing connectors removed
```

The first field decides whether the arguments are database column names (lower case initial)
or Go field names; literal values are escaped. Use `AndArgs` / `OrArgs` / `LikeArgs` (or
`AndWithWhereArgs` / `OrWithWhereArgs`) to get `?` placeholders plus arguments. A field or value
that cannot be rendered fails closed with `1!=1`.

## Helpers

```go
qs, e := mysqlx.InsertMany("user", "(a,b)", "(?,?)", 3)
// qs = INSERT INTO `user` (`a`,`b`) VALUES (?,?),(?,?),(?,?)

mysqlx.In("id", ids)                            // `id` IN (…) / `id`=… / 1!=1
qs, args, e := mysqlx.InValues("code", codes)  // placeholders plus string values
field, args, ok := mysqlx.ArgPairs(condId, argStmts) // partial UPDATE builder
```

`InsertMany` validates table/column identifiers and only permits `?`, `NOW()`,
`CURRENT_TIMESTAMP`, or `DEFAULT` in its value pattern. `InValues` returns placeholders and
arguments; pass both to the database call.

Safe increment helpers build capped `x=x+n` expressions:

```go
mysqlx.SafeIncrementUint16Expression("recmds", -3) // recmds=CASE WHEN recmds-3<=0 THEN 0 ELSE recmds-3 END
mysqlx.SafeDuplicateKeyIncrementClause("tb", []mysqlx.DuplicateKeyIncrement{
	mysqlx.DuplicateKeyUint16Increment("recmds"),
})
```

> 中文提示：无符号字段做自减必须配合 `INSERT IGNORE`，否则负数会直接报错，
> 走不到 `ON DUPLICATE KEY UPDATE` 分支。

## Union helpers (sharded tables)

```go
rows, e := mysqlx.InUnionAllTablesQs(ctx, sdb, qsFormat, ids, partitionTables, func(ptb string, id uint64) []any {...})
rows, e := mysqlx.InUnionAllQs(ctx, sdb, qsFormat, ids, func(id uint64) []any {...})
qs := mysqlx.UnionAllQs(qsFormat, partitionTables, func(ptb string) []any {...})
tables, conds := mysqlx.UnionInUint64s(ids, func(id uint64) string { ... })
```

`InArgs` (`"(...)"`) marks the placeholder that receives the generated `IN` list; the helpers
de-duplicate identical sub-queries, append an inlined `LIMIT n`, and return
`ae.ErrorEmptyInput` for an empty id list.

> 中文提示：`InArgs` 是分表 UNION 查询中占位符标记，最终会被替换成 `IN (1,2,3)`。

## O-R-M

```go
type user struct {
    Id   uint64 `db:"id"`
    Name string `db:"name"`
}
func (user) Table() string         { return "user" }
func (user) Indexes() index.Indexes { return index.NewIndexes(index.Primary("id")) }

orm := sdb.ORM(user{})
e := orm.Alter(ctx, 100, map[string]any{"name": "Aario"})
e = orm.DeletePK(ctx, 100)
e = orm.ExistsPK(ctx, 100)
e = orm.Find(ctx, 100, map[string]any{"id": &id, "name": &name})
```

The entity only needs `Table() string` and `Indexes() index.Indexes` (see
[index-model.md](./index-model.md)).
