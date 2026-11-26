package mongodb

import (
	"strings"

	"github.com/aarioai/aago/aa/ae"
	"github.com/aarioai/aago/aa/atype"
	"github.com/aarioai/aago/driver/index"
	"github.com/aarioai/aago/driver/mongodb/bson2"
	"github.com/aarioai/aago/driver/mongodb/bson3"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type connector bool

const (
	and connector = false
	or  connector = true
)

type filter struct {
	connector connector
	value     any
}

type ORMS struct {
	db         *mongo.Database
	entity     index.Entity
	update     any
	baseFilter any
	filters    []filter
	sort       bson.D
	offset     int64
	limit      int64
	error      *ae.Error
}

func ErrorORM(e *ae.Error) *ORMS {
	return &ORMS{error: e}
}

func ORM(db *mongo.Database, entity index.Entity) *ORMS {
	return &ORMS{
		db:      db,
		entity:  entity,
		filters: make([]filter, 0),
		sort:    make(bson.D, 0),
	}
}

// parseFilter(bson.M)
// parseFilter("filed", 100)
// parseFilter("filed", "=", 100)
// operator:
// = > >= < <= != $eq $gt $gte $lt $lte $ne
// & | ^
// $in $nin $all $size $elemMatch $addToSet $pop $pull $push $pushAll $each $position $slice $sort
func parseFilter(args ...any) (any, *ae.Error) {
	switch len(args) {
	case 0:
		return nil, nil
	case 1:
		return args[0], nil
	case 2:
		field, ok := args[0].(string)
		if !ok {
			return nil, ae.NewError("parseFilter: field must be a string")
		}
		return bson.M{field: args[1]}, nil
	case 3:
		field, fieldOK := args[0].(string)
		operator, operatorOK := args[1].(string)
		if !fieldOK {
			return nil, ae.NewError("parseFilter: field must be a string")
		}
		if !operatorOK {
			return nil, ae.NewError("parseFilter: operator must be a string")
		}
		return bson3.C(field, operator, args[2]), nil
	default:
		return nil, ae.NewError("parseFilter: invalid arguments")
	}
}

func (o *ORMS) WithError(e *ae.Error) *ORMS {
	if o.error == nil {
		o.error = e
	}
	return o
}

// Set records the update/replacement document used by FindOneAndUpdate and FindOneAndReplace.
func (o *ORMS) Set(doc any) *ORMS {
	o.update = doc
	return o
}

func (o *ORMS) Filter() any {
	if len(o.filters) == 0 {
		return o.baseFilter
	}
	result := make([]any, 0, len(o.filters))
	conn := o.filters[0].connector
	values := make([]any, 0, len(o.filters)+1)
	if o.baseFilter != nil {
		values = append(values, o.baseFilter)
	}
	values = append(values, o.filters[0].value)
	for i := 1; i < len(o.filters); i++ {
		f := o.filters[i]
		if conn == f.connector {
			values = append(values, f.value)
			continue
		}
		if conn == and {
			result = append(result, bson2.And(values...))
		} else {
			result = append(result, bson2.Or(values...))
		}
		values = values[:0]
		values = append(values, f.value)
		conn = f.connector
	}

	if len(values) > 0 {
		if conn == and {
			result = append(result, bson2.And(values...))
		} else {
			result = append(result, bson2.Or(values...))
		}
	}
	if len(result) == 1 {
		return result[0]
	}
	return bson2.And(result...)
}

func (o *ORMS) Where(args ...any) *ORMS {
	value, err := parseFilter(args...)
	if err != nil {
		return o.WithError(err)
	}
	o.baseFilter = value
	return o
}

func (o *ORMS) WhereExists(field string) *ORMS {
	o.baseFilter = bson3.Exists(field)
	return o
}

func (o *ORMS) WhereNotExists(field string) *ORMS {
	o.baseFilter = bson3.NotExists(field)
	return o
}

// And
// E.g. Where("a",100).And("b", "nin" bson.A{10,20,30}).And("c", "$all", bson.A{1,2,3})
func (o *ORMS) And(args ...any) *ORMS {
	value, err := parseFilter(args...)
	if err != nil {
		return o.WithError(err)
	}
	o.filters = append(o.filters, filter{and, value})
	return o
}

func (o *ORMS) AndExists(field string) *ORMS {
	value := bson3.Exists(field)
	o.filters = append(o.filters, filter{and, value})
	return o
}

func (o *ORMS) AndNotExists(field string) *ORMS {
	value := bson3.NotExists(field)
	o.filters = append(o.filters, filter{and, value})
	return o
}

func (o *ORMS) Or(args ...any) *ORMS {
	value, err := parseFilter(args...)
	if err != nil {
		return o.WithError(err)
	}
	o.filters = append(o.filters, filter{or, value})
	return o
}

func (o *ORMS) OrExists(field string) *ORMS {
	value := bson3.Exists(field)
	o.filters = append(o.filters, filter{or, value})
	return o
}

func (o *ORMS) NotExists(field string) *ORMS {
	value := bson3.NotExists(field)
	o.filters = append(o.filters, filter{or, value})
	return o
}

func (o *ORMS) DescBy(field string) *ORMS {
	o.sort = append(o.sort, bson.E{Key: field, Value: -1})
	return o
}

func (o *ORMS) AscBy(field string) *ORMS {
	o.sort = append(o.sort, bson.E{Key: field, Value: 1})
	return o
}

// OrderBy
// E.g. OrderBy("id", "DESC", "age", "ASC")
func (o *ORMS) OrderBy(pairs ...string) *ORMS {
	for i := 0; i+1 < len(pairs); i += 2 {
		value := -1
		if strings.EqualFold(pairs[i+1], "ASC") {
			value = 1
		}
		o.sort = append(o.sort, bson.E{Key: pairs[i], Value: value})
	}
	return o
}

func (o *ORMS) Sort(sort ...bson.E) *ORMS {
	o.sort = append(o.sort, sort...)
	return o
}

func (o *ORMS) Limit(offset, limit int64) *ORMS {
	o.offset = offset
	o.limit = limit
	return o
}

func (o *ORMS) LimitN(offset uint, limit uint16) *ORMS {
	o.offset = int64(offset)
	o.limit = int64(limit)
	return o
}

func (o *ORMS) Paging(paging atype.Paging) *ORMS {
	return o.LimitN(paging.Offset, paging.Limit)
}
