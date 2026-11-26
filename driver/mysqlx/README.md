# mysqlx（MySQL 8.0+）说明

```go

type UserEntity struct{
	Id uint `db:"id"`
	Username string `db:"username"`
	Name string `db:"name"`
	Age int `db:"age"`
	CreatedAt atype.Datetime `db:"created_at"`
    UpdatedAt atype.Datetime `db:"updated_at"`
}

func (t UserEntity)Table(){
	return "user"
}
func (t UserEntity)Indexes() index.Indexes {
	return index.NewIndexes(
		index.Primary("id"),
        index.Unique("username"),
    )
}



mysqlx.ORM(db, t).DeleteMany(ctx, "name", "Tom")
mysqlx.ORM(db, t).DeleteOne(ctx, "username", "tom")
mysqlx.ORM(db, t).DeletePK(ctx, 1)
mysqlx.ORM(db, t).ExistsOne(ctx, "username", "tom")
mysqlx.ORM(db, t).ExistsPK(ctx, 1)
mysqlx.ORM(db, t).AlterMany(ctx, "name", "Tom", map[string]any{
	"name":"Thomson",
	"age":18,
	"updated_at":now(),
})
mysqlx.ORM(db, t).AlterOne(ctx,"username", "Tom", map[string]any{
    "name":"Thomson",
    "age":18,
    "updated_at":now(),
})
mysqlx.ORM(db, t).Alter(ctx, 1, map[string]any{
    "name":"Thomson",
    "age":18,
    "updated_at":now(),
})
mysqlx.ORM(db, t).Find(ctx, 1, map[string]any{
	"username", &t.Username,
	"name", &t.Name,
	"age", &t.Age,
	"created_at", &t.CreatedAt,
	"updated_at", &t.UpdatedAt,
})
```