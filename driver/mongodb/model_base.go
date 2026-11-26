package mongodb

import (
	"context"
	"reflect"
	"strings"

	"github.com/aarioai/aago/aa/ae"
	"github.com/aarioai/aago/driver"
	"github.com/aarioai/aago/driver/index"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var ErrMongodbInsertManyMustInTheSameCollection = ae.NewError("mongodb insert many must be in the same collection")
var ErrMongodbMissingUniqueKey = ae.NewError("missing primary or unique key")

func CreateIndexes(ctx context.Context, db *mongo.Database, t index.Entity) *ae.Error {
	coll := db.Collection(t.Table())
	// Creating indexes in MongoDB is an idempotent operation. So running db.names.createIndex({name:1}) would create the index only if it didn't already exist.
	models := index.ToMongoDBIndexModels(t)
	if len(models) == 0 {
		return nil
	}
	_, err := coll.Indexes().CreateMany(ctx, models)
	return driver.NewMongodbError(err)
}

func AggregateRaw(ctx context.Context, db *mongo.Database, t index.Entity, pipeline any, opts ...options.Lister[options.AggregateOptions]) (*mongo.Cursor, *ae.Error) {
	coll := db.Collection(t.Table())
	cursor, err := coll.Aggregate(ctx, pipeline, opts...)
	if err != nil {
		return nil, driver.NewMongodbError(err)
	}
	return cursor, nil
}

func Aggregate(ctx context.Context, results any, db *mongo.Database, t index.Entity, pipeline any, opts ...options.Lister[options.AggregateOptions]) *ae.Error {
	cursor, e := AggregateRaw(ctx, db, t, pipeline, opts...)
	if e != nil {
		return e
	}
	defer cursor.Close(ctx)
	err := cursor.All(ctx, results)
	return driver.NewMongodbError(err)
}

func CountDocuments(ctx context.Context, db *mongo.Database, t index.Entity, filter any, opts ...options.Lister[options.CountOptions]) (int64, *ae.Error) {
	coll := db.Collection(t.Table())
	result, err := coll.CountDocuments(ctx, filter, opts...)
	if err != nil {
		return 0, driver.NewMongodbError(err)
	}
	return result, nil
}

func DeleteOne(ctx context.Context, db *mongo.Database, t index.Entity, filter any, opts ...options.Lister[options.DeleteOneOptions]) (*mongo.DeleteResult, *ae.Error) {
	coll := db.Collection(t.Table())
	result, err := coll.DeleteOne(ctx, filter, opts...)
	if err != nil {
		return nil, driver.NewMongodbError(err)
	}
	return result, nil
}

func DeleteMany(ctx context.Context, db *mongo.Database, t index.Entity, filter any, opts ...options.Lister[options.DeleteManyOptions]) (*mongo.DeleteResult, *ae.Error) {
	coll := db.Collection(t.Table())
	result, err := coll.DeleteMany(ctx, filter, opts...)
	if err != nil {
		return nil, driver.NewMongodbError(err)
	}
	return result, nil
}

func Distinct(ctx context.Context, db *mongo.Database, t index.Entity, field string, filter any, opts ...options.Lister[options.DistinctOptions]) (*mongo.DistinctResult, *ae.Error) {
	coll := db.Collection(t.Table())
	result := coll.Distinct(ctx, field, filter, opts...)
	if err := result.Err(); err != nil {
		return nil, driver.NewMongodbError(err)
	}
	return result, nil
}

func Drop(ctx context.Context, db *mongo.Database, t index.Entity, opts ...options.Lister[options.DropCollectionOptions]) *ae.Error {
	coll := db.Collection(t.Table())
	err := coll.Drop(ctx, opts...)
	return driver.NewMongodbError(err)
}

func EstimatedDocumentCount(ctx context.Context, db *mongo.Database, t index.Entity, opts ...options.Lister[options.EstimatedDocumentCountOptions]) (int64, *ae.Error) {
	coll := db.Collection(t.Table())
	result, err := coll.EstimatedDocumentCount(ctx, opts...)
	if err != nil {
		return 0, driver.NewMongodbError(err)
	}
	return result, nil
}

func FindOneRaw(ctx context.Context, db *mongo.Database, t index.Entity, filter any, opts ...options.Lister[options.FindOneOptions]) (*mongo.SingleResult, *ae.Error) {
	coll := db.Collection(t.Table())
	result := coll.FindOne(ctx, filter, opts...)
	if result.Err() != nil {
		return nil, driver.NewMongodbError(result.Err())
	}
	return result, nil
}

func FindOne(ctx context.Context, result any, db *mongo.Database, t index.Entity, filter any, opts ...options.Lister[options.FindOneOptions]) *ae.Error {
	r, e := FindOneRaw(ctx, db, t, filter, opts...)
	if e != nil {
		return e
	}
	err := r.Decode(result)
	if err != nil {
		return driver.NewMongodbError(err)
	}
	return nil
}

func FindManyRaw(ctx context.Context, db *mongo.Database, t index.Entity, filter any, opts ...options.Lister[options.FindOptions]) (*mongo.Cursor, *ae.Error) {
	coll := db.Collection(t.Table())
	cursor, err := coll.Find(ctx, filter, opts...)
	if err != nil {
		e := driver.NewMongodbError(err)
		if e != nil && e.IsNotFound() {
			e = ae.ErrorNoRowsAvailable
		}
		return nil, e
	}
	return cursor, nil
}

func FindMany(ctx context.Context, results any, db *mongo.Database, t index.Entity, filter any, opts ...options.Lister[options.FindOptions]) *ae.Error {
	cursor, e := FindManyRaw(ctx, db, t, filter, opts...)
	if e != nil {
		return e
	}
	defer cursor.Close(ctx)
	if err := cursor.All(ctx, results); err != nil {
		return driver.NewMongodbError(err)
	}
	if isEmptyResult(results) {
		return ae.ErrorNoRowsAvailable
	}
	return nil
}

// isEmptyResult reports whether a decoded query destination holds no rows.
func isEmptyResult(results any) bool {
	v := reflect.ValueOf(results)
	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return true
		}
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.Slice, reflect.Array, reflect.Map:
		return v.Len() == 0
	}
	return false
}

func FindOneAndDelete(ctx context.Context, db *mongo.Database, t index.Entity, filter any, opts ...options.Lister[options.FindOneAndDeleteOptions]) (*mongo.SingleResult, *ae.Error) {
	coll := db.Collection(t.Table())
	result := coll.FindOneAndDelete(ctx, filter, opts...)
	if err := result.Err(); err != nil {
		return nil, driver.NewMongodbError(err)
	}
	return result, nil
}

func FindOneAndReplace(ctx context.Context, db *mongo.Database, t index.Entity, filter any, replacement any, opts ...options.Lister[options.FindOneAndReplaceOptions]) (*mongo.SingleResult, *ae.Error) {
	coll := db.Collection(t.Table())
	result := coll.FindOneAndReplace(ctx, filter, replacement, opts...)
	if err := result.Err(); err != nil {
		return nil, driver.NewMongodbError(err)
	}
	return result, nil
}

func FindOneAndUpdate(ctx context.Context, db *mongo.Database, t index.Entity, filter any, update any, opts ...options.Lister[options.FindOneAndUpdateOptions]) (*mongo.SingleResult, *ae.Error) {
	coll := db.Collection(t.Table())
	result := coll.FindOneAndUpdate(ctx, filter, update, opts...)
	if err := result.Err(); err != nil {
		return nil, driver.NewMongodbError(err)
	}
	return result, nil
}

func InsertOne(ctx context.Context, db *mongo.Database, t index.Entity, opts ...options.Lister[options.InsertOneOptions]) (*mongo.InsertOneResult, *ae.Error) {
	coll := db.Collection(t.Table())
	result, err := coll.InsertOne(ctx, t, opts...)
	if err != nil {
		return nil, driver.NewMongodbError(err)
	}
	return result, nil
}

func InsertMany(ctx context.Context, db *mongo.Database, ts []index.Entity, opts ...options.Lister[options.InsertManyOptions]) (*mongo.InsertManyResult, *ae.Error) {
	if len(ts) == 0 {
		return nil, ae.ErrorEmptyInput
	}
	table := ts[0].Table()
	for _, t := range ts[1:] {
		if t.Table() != table {
			return nil, ErrMongodbInsertManyMustInTheSameCollection
		}
	}
	coll := db.Collection(table)
	result, err := coll.InsertMany(ctx, ts, opts...)
	if err != nil {
		return nil, driver.NewMongodbError(err)
	}
	return result, nil
}

func ReplaceOne(ctx context.Context, db *mongo.Database, t index.Entity, filter any, opts ...options.Lister[options.ReplaceOptions]) (*mongo.UpdateResult, *ae.Error) {
	coll := db.Collection(t.Table())
	result, err := coll.ReplaceOne(ctx, filter, t, opts...)
	if err != nil {
		return nil, driver.NewMongodbError(err)
	}
	return result, nil
}

func UpdateOne(ctx context.Context, db *mongo.Database, t index.Entity, filter any, update any, opts ...options.Lister[options.UpdateOneOptions]) (*mongo.UpdateResult, *ae.Error) {
	coll := db.Collection(t.Table())
	result, err := coll.UpdateOne(ctx, filter, update, opts...)
	if err != nil {
		return nil, driver.NewMongodbError(err)
	}
	return result, nil
}

func UpdateMany(ctx context.Context, db *mongo.Database, t index.Entity, filter any, update any, opts ...options.Lister[options.UpdateManyOptions]) (*mongo.UpdateResult, *ae.Error) {
	coll := db.Collection(t.Table())
	result, err := coll.UpdateMany(ctx, filter, update, opts...)
	if err != nil {
		return nil, driver.NewMongodbError(err)
	}
	return result, nil
}

func UpsertOne(ctx context.Context, db *mongo.Database, t index.Entity, filter any, update any, opts ...options.Lister[options.UpdateOneOptions]) (*mongo.UpdateResult, *ae.Error) {
	opts = append(opts, options.UpdateOne().SetUpsert(true))
	return UpdateOne(ctx, db, t, filter, update, opts...)
}

func UpsertMany(ctx context.Context, db *mongo.Database, t index.Entity, filter any, update any, opts ...options.Lister[options.UpdateManyOptions]) (*mongo.UpdateResult, *ae.Error) {
	if len(opts) == 0 {
		opts = make([]options.Lister[options.UpdateManyOptions], 0, 1)
	}
	opts = append(opts, options.UpdateMany().SetUpsert(true))
	return UpdateMany(ctx, db, t, filter, update, opts...)
}

// InsertOrUpdate upserts the entity identified by its primary or unique indexes.
// Fields tagged with `options:"no_update"`, the primary key and `created_at` are never overwritten.
//
// InsertOrUpdate 依据主键/唯一索引做 upsert；主键、created_at 及 options:"no_update" 字段不会被覆盖。
func InsertOrUpdate(ctx context.Context, db *mongo.Database, t index.Entity, opts ...options.Lister[options.UpdateOneOptions]) (*mongo.UpdateResult, *ae.Error) {
	uniqueKeys := t.Indexes().List(index.KeyPrimary, index.KeyUnique)
	if len(uniqueKeys) == 0 {
		return nil, ErrMongodbMissingUniqueKey
	}

	rv := reflect.ValueOf(t)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil, ae.ErrorEmptyInput
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil, ErrMongodbMissingUniqueKey
	}
	rt := rv.Type()

	type fieldData struct {
		field     string
		value     any
		noUpdate  bool
		zeroValue bool
	}
	fields := make(map[string]fieldData, rt.NumField())
	for i := 0; i < rt.NumField(); i++ {
		field := bsonFieldName(rt.Field(i))
		if field == "" || field == "-" {
			continue
		}
		value := rv.Field(i).Interface()
		noUpdate := field == "created_at" || strings.Contains(rt.Field(i).Tag.Get("options"), "no_update")
		fields[field] = fieldData{
			field:     field,
			value:     value,
			noUpdate:  noUpdate,
			zeroValue: isEmptyFieldValue(rv.Field(i)),
		}
	}

	// A unique index can only identify a document when every one of its fields is set.
	filterOr := bson.A{}
	for _, uniqueFields := range uniqueKeys {
		conds := bson.D{}
		for _, field := range uniqueFields {
			data, ok := fields[field]
			if !ok || data.zeroValue {
				conds = nil
				break
			}
			conds = append(conds, bson.E{Key: field, Value: data.value})
		}
		if len(conds) == 0 {
			continue
		}
		if len(conds) == 1 {
			filterOr = append(filterOr, conds)
			continue
		}
		filterOr = append(filterOr, bson.D{{Key: "$and", Value: conds}})
	}
	if len(filterOr) == 0 {
		return nil, ErrMongodbMissingUniqueKey
	}

	primary := t.Indexes().Primary()
	sets := bson.D{}
	for _, data := range fields {
		if data.noUpdate {
			continue
		}
		if len(primary) == 1 && data.field == primary[0] {
			continue
		}
		sets = append(sets, bson.E{Key: data.field, Value: data.value})
	}
	if len(sets) == 0 {
		return nil, ae.NewError("mongodb insert or update: no updatable fields")
	}

	var filter any = filterOr[0]
	if len(filterOr) > 1 {
		filter = bson.D{{Key: "$or", Value: filterOr}}
	}

	if len(opts) == 0 {
		opts = make([]options.Lister[options.UpdateOneOptions], 0, 1)
	}
	opts = append(opts, options.UpdateOne().SetUpsert(true))
	return UpdateOne(ctx, db, t, filter, bson.D{{Key: "$set", Value: sets}}, opts...)
}

// bsonFieldName returns the BSON field name of a struct field, or "" when it must be ignored.
func bsonFieldName(f reflect.StructField) string {
	if f.PkgPath != "" { // unexported
		return ""
	}
	tag := f.Tag.Get("bson")
	name, opts, _ := strings.Cut(tag, ",")
	if name == "-" {
		return ""
	}
	if strings.Contains(opts, "inline") { // inlined documents have no single field name
		return ""
	}
	if name == "" {
		return strings.ToLower(f.Name)
	}
	return name
}

func isEmptyFieldValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Pointer:
		return v.IsNil()
	}
	return false
}
