# asql — shared SQL builders

Package: `github.com/aarioai/aago/driver/asql`

`asql` holds everything that `mysqlx` and `postgresx` have in common. Each driver only supplies a
`Dialect`; the statement logic, identifier validation and value escaping live here.

## Dialect

```go
var MySQLDialect = asql.Dialect{
    Quote:        '`',
    MaxIdentPart: 64,
    Marker:       asql.QuestionMark,          // ?
    Bind:         func(int) string { return "?" },
    Inline:       inlineMySQLValue,           // '<escaped>'
    Limit:        limitMySQL,                 // LIMIT offset,limit
    Now:          "now()",
}

var PostgresDialect = asql.Dialect{
    Quote:        '"',
    MaxIdentPart: 63,
    Marker:       asql.DollarMark,            // $1
    Bind:         func(i int) string { return "$" + strconv.Itoa(i) },
    Inline:       inlinePostgresValue,        // E'<escaped>'
    Limit:        limitPostgres,              // LIMIT limit OFFSET offset
    Now:          "CURRENT_TIMESTAMP",
}
```

| Method                          | Purpose                                                    |
|---------------------------------|------------------------------------------------------------|
| `Identifier(name)`              | validate and quote a possibly qualified identifier          |
| `Column(name)`                  | validate and quote a single column (no dot)                 |
| `Placeholders(n)`               | `?,?` or `$1,$2`                                            |
| `InlineArgs(fragment, args, o)` | rewrite the markers of `fragment` into escaped literals     |

`o` is the number of markers rendered before `fragment`; a numbered marker `$n` then consumes
`args[n-o-1]`.

## Builders

| Function / type                                            | Output                                        |
|------------------------------------------------------------|-----------------------------------------------|
| `NewCond(dialect, paging)` / `Cond`                        | `WHERE … ORDER BY … LIMIT …` suffix           |
| `RenderFilterExpr(field, expr, quote, bind, limit)`        | one `FilterExpr` condition plus arguments     |
| `StructFieldSelection(value, fields...)`                   | db tag names / values in the requested order  |
| `StructColumns(dialect, value, fields...)`                 | quoted column list                            |
| `StructConditions(dialect, value, op, like, fields...)`    | `col=? AND col=?` plus arguments              |
| `StructConditionsLiteral(...)`                             | escaped literals, `1!=1` when not renderable  |
| `InsertMany(dialect, table, columns, pattern, num)`        | multi-row INSERT with per-dialect markers     |
| `InUint64Set` / `InUintSet` / `InStringSet`                | `col IN (…)` literals or bound arguments      |
| `ArgStmt`                                                  | assignment type for a partial UPDATE          |
| `ArgPairs(dialect, condID, statements)`                    | `col=?,col2=?` plus arguments                 |
| `AssignmentClause` / `SelectColumns`                       | `ORMS` SET list / SELECT list                 |
| `UnionAllQs` / `UnionAllQuery` / `GroupUnionArgs`          | deduplicated `UNION ALL` query text           |
| `ClampIncrement` / `ClampIncrementClause`                  | capped `x=x±n` and upsert increment expressions |
| `NormalizeWhere` / `WhereClause`                           | connector cleanup and `WHERE` wrapping        |

## Rules

1. **Values are placeholders.** Pass untrusted values as arguments (`?` / `$n`); the literal
   renderers (`Cond.Stmt`, `And`, `Like`, …) escape values but bind markers are preferred.
2. **Fail closed.** Invalid grammar, identifiers, or values render `1!=1` instead of silently
   dropping a condition.
3. **Field names are whitelisted.** Only `[A-Za-z_$][A-Za-z0-9_$]*` identifier parts are
   accepted and every part is quoted.
4. **Dialects own the spelling.** Never hardcode quotes, markers or `LIMIT` syntax in a driver.

## Verification

```text
go test ./driver/asql/...
go vet ./driver/asql/...
```
