# index — entity metadata

Package: `github.com/aarioai/aago/driver/index`

`index` describes how a Go entity maps to a table/collection (name + indexes) and is shared by
`mysqlx` (primary key lookups) and `mongodb` (index creation, upserts).

## Interface

```go
type Entity interface {
    Table() string
    Indexes() Indexes
}
```

## Declaring indexes

```go
type user struct {
    Id        uint64 `bson:"id" db:"id"`
    Email     string `bson:"email" db:"email"`
    CreatedAt int64  `bson:"created_at" db:"created_at"`
}

func (user) Table() string { return "user" }

func (user) Indexes() index.Indexes {
    return index.NewIndexes(
        index.Primary("id"),
        index.Unique("email"),
        index.Index("created_at"),
    )
}
```

| Helper                          | Type                     | SQL          | MongoDB                        |
|---------------------------------|--------------------------|--------------|--------------------------------|
| `Primary`                       | `KeyPrimary`             | `PRIMARY`    | unique                         |
| `Unique`                        | `KeyUnique`              | `u_<fields>` | unique                         |
| `Index`                         | `KeyIndex`               | `i_<fields>` | ascending (`Asc`) / descending |
| `FullText`                      | `KeyText`                | fulltext     | `text` (+ `default_language`)  |
| `Spatial`                       | `KeySpatial`             | spatial      | –                              |
| `Spatial2D` / `Spatial2DSphere` | `KeySpatial2D`/`…Sphere` | –            | `2d` / `2dsphere`              |
| `Hashed`                        | `KeyHashed`              | –            | `hashed`                       |

Index names are generated as a one-character prefix plus the field names with underscores
removed (`u_email`, `i_createdat`), with `PRIMARY` for the primary key. Columns sort descending
unless `Column.Asc` is set.

> 中文提示：索引名自动生成，保证在同一实体内部唯一；`PRIMARY` 固定命名。

## Lookups

```go
pk, e := user{}.Indexes().PrimaryKey()   // "id"; error when missing or composite
fields := user{}.Indexes().Primary()     // []string{"id"}
uniques := user{}.Indexes().List(index.KeyUnique) // map[name][]field
```

`PrimaryKey` returns `ErrMissingPKDefinition` when no primary key exists and
`ErrMultiplePKDefinitions` when the primary key has more than one column.

> 中文提示：复合主键会返回错误，因为 `mysqlx` 的按主键查询只支持单列。

## MongoDB index models

```go
models := index.ToMongoDBIndexModels(user{}) // []mongo.IndexModel
```

Used by `mongodb.CreateIndexes`; creating indexes is idempotent in MongoDB, so calling it on
every start-up is safe.
