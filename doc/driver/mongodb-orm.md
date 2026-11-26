# MongoDB O-R-M

Packages: `github.com/aarioai/aago/driver/mongodb`, `.../mongodb/bson2`,
`.../mongodb/bson3`

## Model entry point

```go
model := mongodb.NewDB(app, "stats")      // app + config section

client, db, e := model.DB()               // pooled client, db = client.Database(dbName)
orm := model.ORM(user{}); orm.Where("status", 0).FindMany(ctx, &users)
```

`Model.ORM` returns `ErrorORM(e)` when the connection cannot be created, so the chain stays
valid and the first terminal call returns the configuration error.

> 中文提示：连接失败时 `ORM` 返回一个带错误的 ORMS，调用方只需检查最终返回值即可。

## Building a query

```go
err := model.ORM(user{}).
    Where("status", 0).          // primary filter
    And("age", ">=", 18).        // comparison
    Or("vip", true).             // OR group
    DescBy("created_at").
    Limit(0, 20).
    FindMany(ctx, &users)
```

| Method                                                                   | Description                                                            |
|--------------------------------------------------------------------------|------------------------------------------------------------------------|
| `Where(args...)`                                                         | base filter (`bson.M`, `(field, value)`, `(field, operator, value)`)   |
| `And/Or(args...)`                                                        | append a filter with the given connector                               |
| `WhereExists/WhereNotExists/AndExists/AndNotExists/OrExists/NotExists`   | `$exists` shortcuts                                                    |
| `AscBy/DescBy/OrderBy("k","ASC",…)`                                      | sorting                                                                |
| `Sort(bson.E...)`                                                        | raw sort entries                                                       |
| `Limit(offset, limit)` / `LimitN(uint, uint16)` / `Paging(atype.Paging)` | pagination                                                             |
| `Set(doc)`                                                               | update/replacement document for `FindOneAndUpdate`/`FindOneAndReplace` |
| `WithError(e)`                                                           | attach the first error                                                 |

`OrderBy` ignores a dangling key (odd number of arguments) instead of panicking.

> 中文提示：`OrderBy("id","DESC","age")` 这种缺参数写法不会再 panic，多余项会被忽略。

## Terminal operations

| Method                                                        | Returns                             |
|---------------------------------------------------------------|-------------------------------------|
| `Insert` / `InsertOne`                                        | `*mongo.InsertOneResult`            |
| `InsertMany` (free function / `Model.InsertMany`)             | `*mongo.InsertManyResult`           |
| `FindOne` / `FindOneRaw`                                      | `*ae.Error` / `*mongo.SingleResult` |
| `FindMany` / `FindManyRaw`                                    | `*ae.Error` / `*mongo.Cursor`       |
| `FindOneAndUpdate` / `FindOneAndReplace` / `FindOneAndDelete` | `*mongo.SingleResult`               |
| `UpdateOne` / `UpdateMany` / `ReplaceOne`                     | `*mongo.UpdateResult`               |
| `UpsertOne` / `UpsertMany` / `InsertOrUpdate`                 | `*mongo.UpdateResult`               |
| `DeleteOne` / `DeleteMany`                                    | `*mongo.DeleteResult`               |
| `CountDocuments` / `EstimatedDocumentCount`                   | `int64`                             |
| `Distinct`                                                    | `*mongo.DistinctResult`             |
| `Aggregate` / `AggregateRaw`                                  | `*ae.Error` / `*mongo.Cursor`       |
| `Drop` / `CreateIndexes`                                      | `*ae.Error`                         |

Behaviour notes:

- `FindMany` returns `ae.ErrorNoRowsAvailable` when nothing matched (slice/map destinations).
- `FindOneAnd*` now return the driver error instead of a `SingleResult` holding it.
- `UpsertOne`/`UpsertMany` append `SetUpsert(true)`; `InsertOrUpdate` builds the filter from the
  entity's primary/unique indexes.

> 中文提示：`FindMany` 无结果时返回 `ae.ErrorNoRowsAvailable`，`FindOneAnd*` 的错误在返回值中直接体现。

## `InsertOrUpdate`

```go
res, e := model.ORM(user{}).InsertOrUpdate(ctx)
```

- The filter is built from every primary/unique index whose fields are all non-zero; several
  usable indexes are combined with `$or`.
- `$set` receives all updatable fields except the primary key, `created_at` and fields tagged
  `options:"no_update"`.
- Returns `mongodb.ErrMongodbMissingUniqueKey` when no usable unique index exists.

> 中文提示：主键、`created_at`、`options:"no_update"` 字段不会被覆盖；没有可用唯一索引时报错。

## `FindOrInsert`

```go
entity, e := mongodb.FindOrInsert(ctx, client, db, user{}, filter, func() index.Entity { return &user{} })
```

Lookup and insert run in one transaction; results are decoded into the entity returned by the
factory (an interface value cannot be decoded into directly).

> 中文提示：查找与插入在同一事务中完成；`newEntity()` 同时负责解码与插入。

## Condition builders

`bson2` (top-level operators) and `bson3` (field scoped) build documents ergonomically:

```go
bson3.C("age", ">=", 18)         // bson.M{"age": bson.M{"$gte": 18}}
bson3.In("status", 0, 1)         // {"status": {"$in": [0,1]}}
bson3.Exists("email")            // {"email": {"$exists": true}}
bson3.Bit("flags", "&", 8)       // {"$bit": {"flags": {"and": 8}}}

bson2.And(filter1, filter2)      // {"$and": [...]}
bson2.Set(bson.M{"a": 1})        // {"$set": {"a": 1}}
bson2.SetOnInsert(bson.M{"a": 1}) // {"$setOnInsert": {"a": 1}}
bson2.Unset("a", "b")            // {"$unset": {"a": "", "b": ""}}
bson2.Inc(bson.M{"score": 1})    // {"$inc": {"score": 1}}
```

`bson3.C` maps the aliases `= != > >= < <=` to `$eq $ne $gt $gte $lt $lte`; bit operators
(`& | ^`) require an integer operand and fall back to the raw operator otherwise.

> 中文提示：`bson3.C("a", ">=", 100)` 现在生成 `{"a": {"$gte": 100}}`（历史版本把别名错写成键名）。
