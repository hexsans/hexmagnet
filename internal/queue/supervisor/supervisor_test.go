package supervisor

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hexsans/hexmagnet/internal/backoff"
	"github.com/hexsans/hexmagnet/internal/queue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestSupervisor_StartsAndStops(t *testing.T) {
	t.Parallel()

	maker := &fakeMaker{}
	backend := newFakeBackend()

	sup := New(Params{
		Maker:   maker.maker(),
		Runtime: backend,
		Topic:   "topic",
		GroupID: "group",
		Handler: func(context.Context, string, []byte) error { return nil },
		Logger:  zap.NewNop().Sugar(),
	})

	hook := sup.Hook()
	require.NoError(t, hook.OnStart(context.Background()))

	waitFor(t, func() bool { return maker.count() == 1 })
	assert.Equal(t, 1, maker.consumer(0).starts())

	require.NoError(t, hook.OnStop(context.Background()))
	assert.Equal(t, 1, maker.consumer(0).stops())
}

func TestSupervisor_BackendChangeRestarts(t *testing.T) {
	t.Parallel()

	maker := &fakeMaker{}
	backend := newFakeBackend()

	sup := New(Params{
		Maker:   maker.maker(),
		Runtime: backend,
		Topic:   "topic",
		GroupID: "group",
		Handler: func(context.Context, string, []byte) error { return nil },
		Logger:  zap.NewNop().Sugar(),
	})

	hook := sup.Hook()
	require.NoError(t, hook.OnStart(context.Background()))

	waitFor(t, func() bool { return maker.count() == 1 })

	backend.notify()

	waitFor(t, func() bool { return maker.count() == 2 })
	assert.Equal(t, 1, maker.consumer(0).stops())
	assert.Equal(t, 1, maker.consumer(1).starts())

	require.NoError(t, hook.OnStop(context.Background()))
}

func TestSupervisor_TriggerEvent(t *testing.T) {
	t.Parallel()

	maker := &fakeMaker{}
	backend := newFakeBackend()

	triggerCh := make(chan struct{}, 1)

	var (
		events  atomic.Int32
		restart atomic.Bool
	)

	sup := New(Params{
		Maker:   maker.maker(),
		Runtime: backend,
		Topic:   "topic",
		GroupID: "group",
		Handler: func(context.Context, string, []byte) error { return nil },
		Trigger: &Trigger{
			Subscribe: func() (<-chan struct{}, func()) {
				return triggerCh, func() {}
			},
			OnEvent: func(context.Context) bool {
				events.Add(1)

				return restart.Load()
			},
		},
		Logger: zap.NewNop().Sugar(),
	})

	hook := sup.Hook()
	require.NoError(t, hook.OnStart(context.Background()))

	waitFor(t, func() bool { return maker.count() == 1 })

	triggerCh <- struct{}{}

	waitFor(t, func() bool { return events.Load() == 1 })

	assert.Equal(t, 1, maker.count(), "event without restart must keep the consumer")

	restart.Store(true)

	triggerCh <- struct{}{}

	waitFor(t, func() bool { return maker.count() == 2 })
	assert.Equal(t, 1, maker.consumer(0).stops())

	require.NoError(t, hook.OnStop(context.Background()))
}

func TestSupervisor_PrepareErrorStopsWithoutConsumer(t *testing.T) {
	t.Parallel()

	maker := &fakeMaker{}
	backend := newFakeBackend()

	sup := New(Params{
		Maker:   maker.maker(),
		Runtime: backend,
		Topic:   "topic",
		GroupID: "group",
		Prepare: func(context.Context) (queue.MessageHandler, error) {
			return nil, errors.New("boom")
		},
		Logger: zap.NewNop().Sugar(),
	})

	hook := sup.Hook()
	require.NoError(t, hook.OnStart(context.Background()))
	require.NoError(t, hook.OnStop(context.Background()))

	assert.Equal(t, 0, maker.count())
}

func TestSupervisor_CreateRetry(t *testing.T) {
	t.Parallel()

	maker := &fakeMaker{}
	maker.failures = 1
	backend := newFakeBackend()

	sup := New(Params{
		Maker:      maker.maker(),
		Runtime:    backend,
		Topic:      "topic",
		GroupID:    "group",
		Handler:    func(context.Context, string, []byte) error { return nil },
		Logger:     zap.NewNop().Sugar(),
		retryDelay: &backoff.Config{Base: time.Millisecond, Max: time.Millisecond},
	})

	hook := sup.Hook()
	require.NoError(t, hook.OnStart(context.Background()))

	waitFor(t, func() bool { return maker.count() == 1 })

	require.NoError(t, hook.OnStop(context.Background()))
}

func TestSupervisor_ShutdownDoesNotWaitForStartContext(t *testing.T) {
	t.Parallel()

	maker := &fakeMaker{}
	backend := newFakeBackend()

	sup := New(Params{
		Maker:   maker.maker(),
		Runtime: backend,
		Topic:   "topic",
		GroupID: "group",
		Handler: func(context.Context, string, []byte) error { return nil },
		Logger:  zap.NewNop().Sugar(),
	})

	hook := sup.Hook()

	// A long-lived start context: the supervisor must not depend on it being
	// canceled to shut down.
	require.NoError(t, hook.OnStart(context.Background()))

	waitFor(t, func() bool { return maker.count() == 1 })

	stopped := make(chan struct{})

	go func() {
		_ = hook.OnStop(context.Background())

		close(stopped)
	}()

	select {
	case <-stopped:
	case <-time.After(5 * time.Second):
		t.Fatal("OnStop blocked waiting for the start context")
	}

	assert.Equal(t, 1, maker.consumer(0).stops())
}

func waitFor(t *testing.T, fn func() bool) {
	t.Helper()

	require.Eventually(t, fn, 5*time.Second, time.Millisecond)
}

type fakeConsumer struct {
	mu      sync.Mutex
	started int
	stopped int
}

func (c *fakeConsumer) Start(context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.started++

	return nil
}

func (c *fakeConsumer) Stop(context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.stopped++

	return nil
}

func (c *fakeConsumer) starts() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.started
}

func (c *fakeConsumer) stops() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.stopped
}

type fakeMaker struct {
	mu        sync.Mutex
	consumers []*fakeConsumer
	failures  int
}

func (m *fakeMaker) maker() queue.ConsumerMaker {
	return queue.ConsumerMaker{
		NewConsumer: func(string, string, queue.MessageHandler, *zap.SugaredLogger) (queue.Consumer, error) {
			m.mu.Lock()
			defer m.mu.Unlock()

			if m.failures > 0 {
				m.failures--

				return nil, errors.New("create failed")
			}

			c := &fakeConsumer{}
			m.consumers = append(m.consumers, c)

			return c, nil
		},
	}
}

func (m *fakeMaker) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return len(m.consumers)
}

func (m *fakeMaker) consumer(i int) *fakeConsumer {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.consumers[i]
}

type fakeBackend struct {
	ch chan struct{}
}

func newFakeBackend() *fakeBackend {
	return &fakeBackend{ch: make(chan struct{}, 8)}
}

func (b *fakeBackend) SubscribeBackendChanges() (<-chan struct{}, func()) {
	return b.ch, func() {}
}

func (b *fakeBackend) notify() {
	b.ch <- struct{}{}
}
