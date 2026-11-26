---
name: aago-mongodb-orm
description: >-
  Use the aago driver MongoDB layer: build a Model from an aago App, chain filters with ORMS,
  choose between Find/Update/Upsert terminals, build BSON with bson2/bson3, and rely on
  index.Entity metadata for index creation and upserts. Use when adding Mongo queries, fixing
  filter documents, or implementing upsert/transaction logic.
---

# MongoDB with aago driver

## When to use this skill

- Adding or changing a MongoDB collection accessor in this repository.
- A filter or update document is wrong (unexpected operators, missing `$gte`, `$unset` shape).
- Implementing "insert or update" / "find or insert" flows.

## Ground rules

1. **Entities describe collections.** Implement `Table() string` and `Indexes() index.Indexes`;
   `Primary`/`Unique` indexes drive `InsertOrUpdate` and `CreateIndexes`.
2. **Use the pool.** `mongodb.NewDB(app, section).DB()` returns the shared client; never
   `Disconnect` it. Close with `driver.CloseMongodbPool(ctx)` at shutdown.
3. **Errors are `*ae.Error`.** `FindMany` yields `ae.ErrorNoRowsAvailable` when nothing matched;
   a missing document is `ae.ErrorNotFound`.
4. **`$set` on upsert is explicit.** Use `ORM(...).Set(doc)` for `FindOneAndUpdate` /
   `FindOneAndReplace`; pass `update` documents to `UpdateOne`/`UpsertOne` directly.

## Building queries

```go
model := mongodb.NewDB(app, "stats")

var users []user
e := model.ORM(user{}).
    Where("status", 0).
    And("age", ">=", 18).
    Or("vip", true).
    DescBy("created_at").
    Limit(0, 20).
    FindMany(ctx, &users)
```

Filters accept `bson.M`, `(field, value)` or `(field, operator, value)`:

```go
Where("age", ">=", 18)        // -> {"age": {"$gte": 18}}
Where("status", 0)            // -> {"status": 0}
Where(bson.M{...})            // raw document
```

`OrderBy("id","DESC","created_at","ASC")` ignores a trailing key with no direction.

## Terminals

| Goal                            | Call                                                             |
|---------------------------------|------------------------------------------------------------------|
| one document                    | `FindOne(ctx, &dst)` / `FindOneRaw(ctx)`                         |
| many documents                  | `FindMany(ctx, &slice)` / `FindManyRaw(ctx)`                     |
| count                           | `CountDocuments(ctx)` / `EstimatedDocumentCount(ctx)`            |
| insert                          | `Insert(ctx)` (entity from `ORM`) or `InsertMany(ctx, entities)` |
| update                          | `UpdateOne(ctx, update)` / `UpdateMany(ctx, update)`             |
| upsert                          | `UpsertOne(ctx, update)` / `UpsertMany(ctx, update)`             |
| insert-or-update by keys        | `InsertOrUpdate(ctx)`                                            |
| find-or-insert in a transaction | `mongodb.FindOrInsert(ctx, client, db, entity, filter, factory)` |
| indexes                         | `CreateIndexes(ctx)` (idempotent)                                |

`InsertOrUpdate` never overwrites the primary key, `created_at`, or fields tagged
`options:"no_update"`, and returns `mongodb.ErrMongodbMissingUniqueKey` when no usable key exists.

## BSON builders

```go
bson3.C("age", ">=", 18)            // {"age": {"$gte": 18}}
bson3.In("status", 0, 1)
bson3.Exists("email")
bson2.And(f1, f2)
bson2.Set(bson.M{"a": 1})
bson2.SetOnInsert(bson.M{"created_at": now})
bson2.Unset("a", "b")               // {"$unset": {"a": "", "b": ""}}
bson2.Inc(bson.M{"score": 1})
```

Aliases accepted by `bson3.C`: `= != > >= < <=` (plus the raw `$eq $ne $gt $gte $lt $lte`).

## Pitfalls

- Do not decode into an `index.Entity` interface value; use a concrete pointer/factory
  (`FindOrInsert` does this for you).
- `FindOneAnd*` return the driver error directly — no need to inspect `SingleResult.Err()`.
- Credentials live in config only; never put them in a URI.

## Verification

- `go test ./driver/mongodb/... ./driver/index/...`, `go vet ./...`.
- Full reference: `../../../../aago/doc/driver/mongodb-orm.md` and `../../../../aago/doc/driver/index-model.md`.
