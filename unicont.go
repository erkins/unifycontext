package unicont

import (
	"context"
	"sync"
	"time"
)

type Cancel struct {
}

func (c *Cancel) Error() string {
	return "context canceled"
}

type unicont struct {
	ctx        context.Context
	ctxs       []context.Context
	done       chan struct{}
	err        error
	errMu      sync.Mutex
	cancelFunc context.CancelFunc
	cancelCtx  context.Context
}

func Unify(ctx context.Context, ctxs ...context.Context) (context.Context, context.CancelFunc) {
	cancelCtx, cancelFunc := context.WithCancel(context.Background())
	u := &unicont{
		done:       make(chan struct{}),
		ctx:        ctx,
		ctxs:       ctxs,
		cancelCtx:  cancelCtx,
		cancelFunc: cancelFunc,
	}

	go func() {
		once := sync.Once{}

		if len(u.ctxs) == 1 {
			go func() {
				select {
				case <-u.cancelCtx.Done():
					u.cancel(&Cancel{})
				case <-u.ctx.Done():
					u.cancel(u.ctx.Err())
				case <-u.ctxs[0].Done():
					u.cancel(u.ctxs[0].Err())
				}
			}()
			return
		}

		go func() {
			select {
			case <-u.cancelCtx.Done():
				once.Do(func() {
					u.cancel(&Cancel{})
				})
			case <-ctx.Done():
				once.Do(func() {
					u.cancel(ctx.Err())
				})
			}
		}()

		for _, ctx := range u.ctxs {
			go func() {
				select {
				case <-u.cancelCtx.Done():
					once.Do(func() {
						u.cancel(&Cancel{})
					})
				case <-ctx.Done():
					once.Do(func() {
						u.cancel(ctx.Err())
					})
				}
			}()
		}
	}()

	return u, cancelFunc
}

func (u *unicont) Deadline() (time.Time, bool) {
	min := time.Time{}

	if deadline, ok := u.ctx.Deadline(); ok {
		min = deadline
	}

	for _, ctx := range u.ctxs {
		if deadline, ok := ctx.Deadline(); ok {
			if min.IsZero() || deadline.Before(min) {
				min = deadline
			}
		}
	}

	return min, !min.IsZero()
}

func (u *unicont) Done() <-chan struct{} {
	return u.done
}

func (u *unicont) Err() error {
	u.errMu.Lock()
	defer u.errMu.Unlock()
	return u.err
}

func (u *unicont) Value(key interface{}) interface{} {
	if value := u.ctx.Value(key); value != nil {
		return value
	}

	for _, ctx := range u.ctxs {
		if value := ctx.Value(key); value != nil {
			return value
		}
	}

	return nil
}

func (u *unicont) cancel(err error) {
	u.cancelFunc()
	u.errMu.Lock()
	u.err = err
	u.errMu.Unlock()
	u.done <- struct{}{}
}
