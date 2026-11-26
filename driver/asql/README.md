# ASQL

用于客户端查询服务端的仿 SQL 语句。

该包同时提供 mysqlx / postgresx 的公共构造器：`Dialect`、`Cond`、结构体条件、INSERT/IN/UNION
构造器以及带上下限的增量表达式。驱动包只负责提供各自方言（标识符引号、占位符、`LIMIT` 语法）。

完整说明见 [doc/asql.md](../../doc/asql.md)。

```go
cond := asql.NewCond(&asql.PostgresDialect, paging)
cond.And("u.id", "100").OrderBy("u.id DESC")
suffix, args := cond.StmtArgs() // WHERE "u"."id"=$1 ORDER BY "u"."id" DESC LIMIT 10 OFFSET 0
```
