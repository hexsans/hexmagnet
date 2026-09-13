package memoryqueue

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hexsans/hexmagnet/internal/backoff"
	"github.com/hexsans/hexmagnet/internal/queue/delivery"
	"github.com/hexsans/hexmagnet/internal/queue/permanent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNew(t *testing.T) {
	t.Parallel()

	mq := New(zap.NewNop().Sugar())
	assert.NotNil(t, mq)
	assert.NotNil(t, mq.topics)
	assert.NotNil(t, mq.offsets)
}

func TestProduce(t *testing.T) {
	t.Parallel()

	mq := New(zap.NewNop().Sugar())
	mq.Produce("topic-a", "key-1", "value-1")

	mq.mu.RLock()
	ts, ok := mq.topics["topic-a"]
	mq.mu.RUnlock()
	require.True(t, ok)
	assert.Equal(t, uint64(1), ts.NextSeq)
	assert.Equal(t, 1, ts.Tail)

	mq.mu.RLock()

	msg := ts.Messages[0]

	mq.mu.RUnlock()
	require.NotNil(t, msg)
	assert.Equal(t, uint64(0), msg.Seq)
	assert.Equal(t, "key-1", msg.Key)
}

func TestProduce_AfterClose(t *testing.T) {
	t.Parallel()

	mq := New(zap.NewNop().Sugar())
	require.NoError(t, mq.Close(context.Background()))

	mq.Produce("topic-a", "key-1", "value-1")

	mq.mu.RLock()
	_, ok := mq.topics["topic-a"]
	mq.mu.RUnlock()
	assert.False(t, ok, "should not create topic after close")
}

func TestProduce_InvalidJSON(t *testing.T) {
	t.Parallel()

	mq := New(zap.NewNop().Sugar())
	mq.Produce("topic-a", "key", make(chan int))

	mq.mu.RLock()
	_, ok := mq.topics["topic-a"]
	mq.mu.RUnlock()
	assert.False(t, ok, "should not create topic on marshal error")
}

func TestNewConsumer(t *testing.T) {
	t.Parallel()

	mq := New(zap.NewNop().Sugar())
	c := mq.NewConsumer("topic-a", "group-1", func(_ context.Context, _ string, _ []byte) error {
		return nil
	}, zap.NewNop().Sugar())
	assert.NotNil(t, c)
	assert.Equal(t, "topic-a", c.topic)
	assert.Equal(t, "group-1", c.groupID)
	assert.NotNil(t, c.ch)

	mq.mu.RLock()
	ts := mq.topics["topic-a"]
	mq.mu.RUnlock()
	require.NotNil(t, ts)
	require.Len(t, ts.Consumers, 1)
	assert.Equal(t, "group-1", ts.Consumers[0].GroupID)
}

func TestConsumer_StartStop(t *testing.T) {
	t.Parallel()

	mq := New(zap.NewNop().Sugar())

	received := make(chan struct{}, 1)
	c := mq.NewConsumer("topic-a", "group-1", func(_ context.Context, key string, _ []byte) error {
		assert.Equal(t, "k1", key)

		received <- struct{}{}

		return nil
	}, zap.NewNop().Sugar())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := c.Start(ctx)
	require.NoError(t, err)

	mq.Produce("topic-a", "k1", "v1")

	select {
	case <-received:
	case <-ctx.Done():
		t.Fatal("timeout waiting for handler")
	}

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer stopCancel()

	require.NoError(t, c.Stop(stopCtx))
}

func TestConsumer_DoubleStart(t *testing.T) {
	t.Parallel()

	mq := New(zap.NewNop().Sugar())
	c := mq.NewConsumer("t", "g", func(_ context.Context, _ string, _ []byte) error {
		return nil
	}, zap.NewNop().Sugar())

	ctx := context.Background()
	require.NoError(t, c.Start(ctx))
	assert.True(t, c.started.Load())
	require.NoError(t, c.Start(ctx))
	assert.True(t, c.started.Load())
	require.NoError(t, c.Stop(ctx))
}

func TestConsumer_ContextCancelled(t *testing.T) {
	t.Parallel()

	mq := New(zap.NewNop().Sugar())
	c := mq.NewConsumer("t", "g", func(_ context.Context, _ string, _ []byte) error {
		return nil
	}, zap.NewNop().Sugar())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.NoError(t, c.Start(ctx))

	c.wg.Wait()
	require.NoError(t, c.Stop(context.Background()))
}

func TestConsumer_HandlerError(t *testing.T) {
	t.Parallel()

	mq := New(zap.NewNop().Sugar())
	c := mq.NewConsumer("t", "g", func(_ context.Context, _ string, _ []byte) error {
		return assert.AnError
	}, zap.NewNop().Sugar())

	ctx := context.Background()
	require.NoError(t, c.Start(ctx))

	mq.Produce("t", "k", "v")
	time.Sleep(50 * time.Millisecond)
	require.NoError(t, c.Stop(ctx))
}

func TestConsumer_PermanentErrorAdvancesOffset(t *testing.T) {
	t.Parallel()

	mq := New(zap.NewNop().Sugar())
	mq.SetDelivery(delivery.Config{
		MaxAttempts: 2,
		Backoff:     backoff.Config{Base: time.Millisecond},
	})

	handled := make(chan struct{}, 10)
	c := mq.NewConsumer("t", "g", func(_ context.Context, key string, _ []byte) error {
		handled <- struct{}{}

		switch key {
		case "permanent":
			return permanent.Mark(assert.AnError)
		case "transient":
			return assert.AnError
		default:
			return nil
		}
	}, zap.NewNop().Sugar())

	ctx := context.Background()
	require.NoError(t, c.Start(ctx))

	mq.Produce("t", "ok", "v")
	mq.Produce("t", "permanent", "v")
	mq.Produce("t", "transient", "v")

	// ok once, permanent once, transient retried MaxAttempts times.
	for range 4 {
		select {
		case <-handled:
		case <-time.After(5 * time.Second):
			t.Fatal("timeout waiting for handler")
		}
	}

	require.NoError(t, c.Stop(ctx))

	mq.mu.RLock()
	offset := mq.offsets["g"]["t"]
	mq.mu.RUnlock()

	// All three messages advance the offset (next seq): the transient message
	// is retried until MaxAttempts and then discarded as poison.
	assert.Equal(t, uint64(3), offset)
}

func TestConsumer_RetryableErrorDiscardedAfterMaxAttempts(t *testing.T) {
	t.Parallel()

	mq := New(zap.NewNop().Sugar())
	mq.SetDelivery(delivery.Config{
		MaxAttempts: 3,
		Backoff:     backoff.Config{Base: time.Millisecond},
	})

	var attempts atomic.Int32

	c := mq.NewConsumer("t", "g", func(_ context.Context, _ string, _ []byte) error {
		attempts.Add(1)

		return assert.AnError
	}, zap.NewNop().Sugar())

	ctx := context.Background()
	require.NoError(t, c.Start(ctx))

	mq.Produce("t", "k", "v")

	require.Eventually(t, func() bool {
		mq.mu.RLock()
		defer mq.mu.RUnlock()

		offset, ok := mq.offsets["g"]["t"]

		return ok && offset == 1
	}, 5*time.Second, 5*time.Millisecond, "poison message should be discarded and the offset advanced")

	assert.Equal(t, int32(3), attempts.Load())

	require.NoError(t, c.Stop(ctx))
}

func TestConsumer_CanceledErrorRetriesWhileContextAlive(t *testing.T) {
	t.Parallel()

	mq := New(zap.NewNop().Sugar())
	mq.SetDelivery(delivery.Config{
		MaxAttempts: 2,
		Backoff:     backoff.Config{Base: time.Millisecond},
	})

	var attempts atomic.Int32

	c := mq.NewConsumer("t", "g", func(_ context.Context, _ string, _ []byte) error {
		attempts.Add(1)

		return context.Canceled
	}, zap.NewNop().Sugar())

	require.NoError(t, c.Start(context.Background()))

	mq.Produce("t", "k", "v")

	require.Eventually(t, func() bool {
		mq.mu.RLock()
		defer mq.mu.RUnlock()

		offset, ok := mq.offsets["g"]["t"]

		return ok && offset == 1
	}, 5*time.Second, 5*time.Millisecond, "canceled error with a live context must be treated as retryable")

	assert.Equal(t, int32(2), attempts.Load())

	require.NoError(t, c.Stop(context.Background()))
}

func TestConsumer_ContextCancelledStopsWithoutAdvancing(t *testing.T) {
	t.Parallel()

	mq := New(zap.NewNop().Sugar())

	started := make(chan struct{}, 1)

	c := mq.NewConsumer("t", "g", func(ctx context.Context, _ string, _ []byte) error {
		started <- struct{}{}

		<-ctx.Done()

		return ctx.Err()
	}, zap.NewNop().Sugar())

	ctx, cancel := context.WithCancel(context.Background())

	require.NoError(t, c.Start(ctx))

	mq.Produce("t", "k", "v")

	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for handler")
	}

	cancel()
	c.wg.Wait()

	mq.mu.RLock()
	_, ok := mq.offsets["g"]["t"]
	mq.mu.RUnlock()

	assert.False(t, ok, "message must not be acknowledged when the context is canceled")

	require.NoError(t, c.Stop(context.Background()))
}

func TestConsumer_UnackedMessageRedeliveredOnRestart(t *testing.T) {
	t.Parallel()

	mq := New(zap.NewNop().Sugar())

	c1 := mq.NewConsumer("t", "g", func(ctx context.Context, _ string, _ []byte) error {
		<-ctx.Done()

		return ctx.Err()
	}, zap.NewNop().Sugar())

	ctx, cancel := context.WithCancel(context.Background())

	require.NoError(t, c1.Start(ctx))

	mq.Produce("t", "k", "v")

	require.Eventually(t, func() bool {
		mq.mu.RLock()
		defer mq.mu.RUnlock()

		ts, ok := mq.topics["t"]

		return ok && ts.Tail != ts.Head
	}, 5*time.Second, 5*time.Millisecond, "message should be pending while the handler blocks")

	cancel()
	c1.wg.Wait()
	require.NoError(t, c1.Stop(context.Background()))

	redelivered := make(chan struct{}, 1)

	c2 := mq.NewConsumer("t", "g", func(_ context.Context, _ string, _ []byte) error {
		redelivered <- struct{}{}

		return nil
	}, zap.NewNop().Sugar())

	require.NoError(t, c2.Start(context.Background()))

	select {
	case <-redelivered:
	case <-time.After(5 * time.Second):
		t.Fatal("unacknowledged message was not redelivered after restart")
	}

	require.NoError(t, c2.Stop(context.Background()))
}

func TestListConsumerGroups_Empty(t *testing.T) {
	t.Parallel()

	mq := New(zap.NewNop().Sugar())
	groups, err := mq.ListConsumerGroups()
	require.NoError(t, err)
	assert.Empty(t, groups)
}

func TestListConsumerGroups_WithConsumers(t *testing.T) {
	t.Parallel()

	mq := New(zap.NewNop().Sugar())
	mq.NewConsumer("topic-a", "group-1", nil, zap.NewNop().Sugar())
	mq.NewConsumer("topic-b", "group-2", nil, zap.NewNop().Sugar())

	groups, err := mq.ListConsumerGroups()
	require.NoError(t, err)
	require.Len(t, groups, 2)
	assert.Equal(t, "group-1", groups[0].GroupID)
	assert.Equal(t, "group-2", groups[1].GroupID)
	assert.Equal(t, "Stable", groups[0].State)
}

func TestDescribeConsumerGroup(t *testing.T) {
	t.Parallel()

	mq := New(zap.NewNop().Sugar())
	mq.NewConsumer("topic-a", "group-1", nil, zap.NewNop().Sugar())
	mq.Produce("topic-a", "k", "v")
	mq.Produce("topic-a", "k2", "v2")

	detail, err := mq.DescribeConsumerGroup("group-1")
	require.NoError(t, err)
	assert.Equal(t, "group-1", detail.GroupID)
	assert.Equal(t, "Stable", detail.State)
	require.Len(t, detail.Topics, 1)
	assert.Equal(t, "topic-a", detail.Topics[0].Topic)
	require.Len(t, detail.Topics[0].Partitions, 1)
	assert.Equal(t, int32(0), detail.Topics[0].Partitions[0].Partition)
	assert.Equal(t, int64(0), detail.Topics[0].Partitions[0].CurrentOffset)
	assert.Equal(t, int64(2), detail.Topics[0].Partitions[0].EndOffset)
	assert.Equal(t, int64(2), detail.Topics[0].Partitions[0].Lag)
}

func TestDescribeConsumerGroup_NotFound(t *testing.T) {
	t.Parallel()

	mq := New(zap.NewNop().Sugar())
	detail, err := mq.DescribeConsumerGroup("nonexistent")
	require.NoError(t, err)
	assert.Equal(t, "nonexistent", detail.GroupID)
	assert.Empty(t, detail.Topics)
}
