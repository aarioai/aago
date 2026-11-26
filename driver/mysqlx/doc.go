// Package mysqlx provides MySQL 8.0+ helpers on top of database/sql:
// pooled DB/Tx execution with *ae.Error results, condition builders
// (Cond with asql.FilterExpr and struct driven fragments), safe increment
// expressions, a small O-R-M and sharded-table UNION helpers.
//
// Every statement builder is shared with postgresx through driver/asql; mysqlx
// only supplies the MySQL dialect (back-quoted identifiers, "?" markers and
// "LIMIT offset,limit"). MySQL 8.0+ is required; there is intentionally no
// MySQL 5.7 fallback.
package mysqlx
