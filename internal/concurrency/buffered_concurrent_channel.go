package concurrency

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"
	"sync/atomic"

	"golang.org/x/sync/semaphore"
)

type BufferedConcurrentChannel[T any] interface {
	In() chan<- T
	Run(context.Context, func(T)) error
	SetConcurrency(n int)
}

func NewBufferedConcurrentChannel[T any](capacity int, concurrency int) BufferedConcurrentChannel[T] {
	c := &bufferedConcurrentChannel[T]{
		ch: make(chan T, capacity),
	}
	c.sem.Store(semaphore.NewWeighted(int64(concurrency)))

	return c
}

type bufferedConcurrentChannel[T any] struct {
	ch  chan T
	sem atomic.Pointer[semaphore.Weighted]
}

func (ch *bufferedConcurrentChannel[T]) In() chan<- T {
	return ch.ch
}

func (ch *bufferedConcurrentChannel[T]) SetConcurrency(n int) {
	ch.sem.Store(semaphore.NewWeighted(int64(n)))
}

func (ch *bufferedConcurrentChannel[T]) Run(ctx context.Context, f func(T)) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case next := <-ch.ch:
			sem := ch.sem.Load()
			if err := sem.Acquire(ctx, 1); err != nil {
				return err
			}

			go func() {
				defer func() {
					if r := recover(); r != nil {
						_, _ = fmt.Fprintf(
							os.Stderr,
							"PANIC in BufferedConcurrentChannel callback: %v\n%s\n",
							r,
							debug.Stack(),
						)
					}
				}()
				defer sem.Release(1)

				f(next)
			}()
		}
	}
}
