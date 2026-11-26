package redisx_test

import (
	"context"
	"testing"

	"github.com/aarioai/aago/aa/ae"
	"github.com/aarioai/aago/driver/redisx"
)

type hashStruct struct {
	Key1 string `redis:"key1"`
	Key2 int    `redis:"key2"`
	Skip string `redis:"-"`
}

// HSet must reject invalid arguments before touching the client,
// so a nil client is safe here.
func TestHSetInvalidArguments(t *testing.T) {
	ctx := context.Background()

	if err := redisx.HSet(ctx, nil, 0, "k"); err != ae.ErrInvalidInput {
		t.Errorf("HSet(no data) = %v, want ErrInvalidInput", err)
	}
	if err := redisx.HSet(ctx, nil, 0, "k", "field"); err != redisx.ErrInvalidHashArgs {
		t.Errorf("HSet(odd args) = %v, want ErrInvalidHashArgs", err)
	}
	if err := redisx.HSet(ctx, nil, 0, "k", 100); err != redisx.ErrInvalidHashArgs {
		t.Errorf("HSet(int) = %v, want ErrInvalidHashArgs", err)
	}
	var nilStruct *hashStruct
	if err := redisx.HSet(ctx, nil, 0, "k", nilStruct); err != ae.ErrInvalidInput {
		t.Errorf("HSet(nil struct pointer) = %v, want ErrInvalidInput", err)
	}
	if err := redisx.HSet(ctx, nil, 0, "k", struct{}{}); err != redisx.ErrInvalidHashArgs {
		t.Errorf("HSet(struct without tags) = %v, want ErrInvalidHashArgs", err)
	}
}
