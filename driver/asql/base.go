package asql

import "github.com/aarioai/aago/aa/ae"

func badSqlError(msg string) *ae.Error {
	return ae.New(ae.InternalServerError, msg)
}
