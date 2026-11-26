package mongodb

import (
	"context"
	"errors"

	"github.com/aarioai/aago/aa/ae"
	"github.com/aarioai/aago/driver"
	"github.com/aarioai/aago/driver/index"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// FindOrInsert finds the first document matching filter, or inserts the entity built by newEntity.
//
// The lookup and the insert run inside one transaction, and results are decoded into a
// fresh entity created by newEntity (an interface value cannot be decoded into directly).
//
// FindOrInsert 在同一事务中查找或插入；newEntity 同时用于结果解码和插入。
func FindOrInsert(ctx context.Context, client *mongo.Client, db *mongo.Database,
	t index.Entity, filter any, newEntity func() index.Entity) (*index.Entity, *ae.Error) {
	sess, err := client.StartSession()
	if err != nil {
		return nil, driver.NewMongodbError(err)
	}
	defer sess.EndSession(ctx)

	var result index.Entity
	var innerErr *ae.Error
	_, err = sess.WithTransaction(ctx, func(sc context.Context) (any, error) {
		innerErr = nil
		found := newEntity()
		e := FindOne(sc, found, db, t, filter)
		if e == nil {
			result = found
			return nil, nil
		}
		if !e.IsNotFound() {
			innerErr = e
			return nil, errors.New(e.Msg)
		}

		created := newEntity()
		if _, e = InsertOne(sc, db, created); e != nil {
			innerErr = e
			return nil, errors.New(e.Msg)
		}
		result = created
		return nil, nil
	})
	if innerErr != nil {
		return nil, innerErr
	}
	if err != nil {
		return nil, driver.NewMongodbError(err)
	}
	if result == nil {
		return nil, driver.NewMongodbError(mongo.ErrNoDocuments)
	}
	return &result, nil
}
