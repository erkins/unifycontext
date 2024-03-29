package unicont

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type key int

const (
	foo key = iota
	bar
	baz
)

func eventually(ch <-chan struct{}) bool {
	timeout, cancelFunc := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancelFunc()

	select {
	case <-ch:
		return true
	case <-timeout.Done():
		return false
	}
}

func Test_Mix_Nominal(t *testing.T) {
	ctx1, cancel1 := context.WithCancel(context.WithValue(context.Background(), foo, "foo"))
	defer cancel1()
	ctx2, cancel2 := context.WithCancel(context.WithValue(context.Background(), bar, "bar"))

	ctx, _ := Unify(ctx1, ctx2)

	deadline, ok := ctx.Deadline()
	assert.True(t, deadline.IsZero())
	assert.False(t, ok)

	assert.Equal(t, "foo", ctx.Value(foo))
	assert.Equal(t, "bar", ctx.Value(bar))
	assert.Nil(t, ctx.Value(baz))

	assert.False(t, eventually(ctx.Done()))
	assert.NoError(t, ctx.Err())

	cancel2()
	assert.True(t, eventually(ctx.Done()))
	assert.Error(t, ctx.Err())
}

func Test_Mix_Deadline_Context1(t *testing.T) {
	ctx1, cancel1 := context.WithTimeout(context.Background(), time.Second)
	defer cancel1()
	ctx2 := context.Background()

	ctx, _ := Unify(ctx1, ctx2)

	deadline, ok := ctx.Deadline()
	assert.False(t, deadline.IsZero())
	assert.True(t, ok)
}

func Test_Mix_Deadline_Context2(t *testing.T) {
	ctx1 := context.Background()
	ctx2, cancel2 := context.WithTimeout(context.Background(), time.Second)
	defer cancel2()

	ctx, _ := Unify(ctx1, ctx2)

	deadline, ok := ctx.Deadline()
	assert.False(t, deadline.IsZero())
	assert.True(t, ok)
}

func Test_Mix_Deadline_None(t *testing.T) {
	ctx, _ := Unify(context.Background(), context.Background())

	deadline, ok := ctx.Deadline()
	assert.True(t, deadline.IsZero())
	assert.False(t, ok)
}

func Test_Cancel(t *testing.T) {
	ctx, cancel := Unify(context.Background(), context.Background())

	cancel()
	assert.True(t, eventually(ctx.Done()))
	assert.Error(t, ctx.Err())
	assert.Equal(t, "context canceled", ctx.Err().Error())
	assert.IsType(t, &Cancel{}, ctx.Err())
}

func Test_Cancel_Multiple(t *testing.T) {
	ctx, cancel := Unify(context.Background(), context.Background(), context.Background())

	cancel()
	assert.True(t, eventually(ctx.Done()))
	assert.Error(t, ctx.Err())
	assert.Equal(t, "context canceled", ctx.Err().Error())
	assert.IsType(t, &Cancel{}, ctx.Err())
}
