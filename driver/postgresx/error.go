package postgresx

import (
	"github.com/aarioai/aago/aa/ae"
	"github.com/aarioai/aago/driver"
)

func newPostgresError(err error, details ...any) *ae.Error {
	return driver.NewPostgresError(err, details...)
}
