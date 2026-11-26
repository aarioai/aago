package index

import (
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// ToMongoDBIndexModels
// @doc https://www.mongodb.com/zh-cn/docs/manual/indexes/
func ToMongoDBIndexModels(t Entity) []mongo.IndexModel {
	indexes := t.Indexes()
	if len(indexes) == 0 {
		return nil
	}
	models := make([]mongo.IndexModel, 0, len(indexes))
	for name, columns := range indexes {
		if len(columns) == 0 {
			continue
		}
		opt := options.Index().SetName(name)
		indexType := columns[0].Type
		switch indexType {
		case KeyPrimary, KeyUnique:
			opt.SetUnique(true)
		}
		keys := make(bson.D, 0, len(columns))
		for _, col := range columns {
			switch col.Type {
			case KeyText:
				keys = append(keys, bson.E{Key: col.Field, Value: "text"})
				if col.Language != "" {
					keys = append(keys, bson.E{Key: "default_language", Value: col.Language})
				}
			case KeyHashed:
				keys = append(keys, bson.E{Key: col.Field, Value: "hashed"})
			case KeySpatial2D:
				keys = append(keys, bson.E{Key: col.Field, Value: "2d"})
			case KeySpatial2DSphere:
				keys = append(keys, bson.E{Key: col.Field, Value: "2dsphere"})

			default:
				value := -1
				if col.Asc {
					value = 1
				}
				keys = append(keys, bson.E{Key: col.Field, Value: value})
			}
		}
		indexModel := mongo.IndexModel{Keys: keys, Options: opt}
		models = append(models, indexModel)
	}
	if len(models) == 0 {
		return nil
	}
	return models
}
