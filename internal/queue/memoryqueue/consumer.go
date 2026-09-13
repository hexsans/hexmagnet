package memoryqueue

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/hexsans/hexmagnet/internal/queue/permanent"
	"github.com/hexsans/hexmagnet/internal/utils"
	"go.uber.org/zap"
)

type MessageHandler func(ctx context.Context, key string, value []byte) error

type Consumer struct {
	queue       *MemoryQueue
	topic       string
	groupID     string
	handler     MessageHandler
	ch          chan *Message
	cancel      context.CancelFunc
	internalCtx context.Context
	wg          sync.WaitGroup
	started     atomic.Bool
	logger      *zap.SugaredLogger
}

func (mq *MemoryQueue) NewConsumer(topic, groupID string, handler MessageHandler, logger *zap.SugaredLogger) *Consumer {
	ctx, cancel := context.WithCancel(context.Background())

	c := &Consumer{
		queue:       mq,
		topic:       topic,
		groupID:     groupID,
		handler:     handler,
		ch:          make(chan *Message, 10_000),
		cancel:      cancel,
		internalCtx: ctx,
		logger:      logger.Named("memqueue_consumer"),
	}

	mq.mu.Lock()
	ts := mq.getOrCreateTopic(topic)

	var offset uint64
	if groupOffsets, ok := mq.offsets[groupID]; ok {
		offset = groupOffsets[topic]
	}

	ts.Consumers = append(ts.Consumers, &consumerState{
		GroupID: groupID,
		Offset:  offset,
		Ch:      c.ch,
		Cancel:  cancel,
	})
	mq.mu.Unlock()

	return c
}

func (c *Consumer) Start(ctx context.Context) error {
	if !c.started.CompareAndSwap(false, true) {
		return nil
	}

	c.wg.Add(1)

	go func() {
		defer utils.Recover(c.logger, "memory queue consumer panicked", "topic", c.topic, "group", c.groupID)

		c.run(ctx)
	}()

	return nil
}

func (c *Consumer) run(ctx context.Context) {
	defer c.wg.Done()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Stop automatically cancels internalCtx; propagate that to the retry
	// waits so shutdown never blocks on a pending backoff.
	stopWatch := context.AfterFunc(c.internalCtx, cancel) //nolint:contextcheck // internalCtx is the consumer-owned shutdown context
	defer stopWatch()

	c.replay(ctx)

	for {
		select {
		case msg, ok := <-c.ch:
			if !ok {
				return
			}

			if !c.process(ctx, msg) {
				return
			}
		case <-c.internalCtx.Done():
			return
		case <-ctx.Done():
			return
		}
	}
}

func (c *Consumer) replay(ctx context.Context) {
	c.queue.mu.RLock()

	ts, ok := c.queue.topics[c.topic]
	if !ok {
		c.queue.mu.RUnlock()
		return
	}

	offset := c.queue.offsets[c.groupID][c.topic]

	count := (ts.Tail - ts.Head + ts.Capacity) % ts.Capacity

	replayMsgs := make([]*Message, 0, count)
	for i := range count {
		idx := (ts.Head + i) % ts.Capacity

		msg := ts.Messages[idx]
		if msg != nil && msg.Seq >= offset {
			replayMsgs = append(replayMsgs, msg)
		}
	}

	c.queue.mu.RUnlock()

	for _, msg := range replayMsgs {
		select {
		case <-ctx.Done():
			return
		default:
		}

		c.process(ctx, msg)
	}
}

// process handles one message with at-least-once semantics: retryable
// failures are retried in place with backoff until MaxAttempts is reached,
// then discarded as poison. It reports whether the consumer loop should keep
// running.
func (c *Consumer) process(ctx context.Context, msg *Message) bool {
	if c.alreadyProcessed(msg) {
		return true
	}

	deliveryCfg := c.queue.deliveryConfig()

	for attempt := 1; ; attempt++ {
		err := c.handler(ctx, msg.Key, []byte(msg.Value))
		if err == nil {
			c.advanceOffset(msg)

			return true
		}

		if ctx.Err() != nil {
			return false
		}

		if permanent.Is(err) {
			c.logger.Debugw("skipping permanently failed message",
				"topic", c.topic, "group", c.groupID, "seq", msg.Seq, "error", err)

			c.advanceOffset(msg)

			return true
		}

		if attempt >= deliveryCfg.MaxAttempts {
			c.logger.Errorw("queue message exceeded delivery attempts, discarding",
				"topic", c.topic, "group", c.groupID, "seq", msg.Seq,
				"attempts", attempt, "error", err)

			c.advanceOffset(msg)

			return true
		}

		c.logger.Warnw("message handler failed, retrying",
			"topic", c.topic, "group", c.groupID, "seq", msg.Seq,
			"attempt", attempt, "error", err)

		if deliveryCfg.Backoff.Wait(ctx, attempt) != nil {
			return false
		}
	}
}

// alreadyProcessed reports whether msg was acknowledged by an earlier
// delivery (replay and the live channel can both surface the same message).
func (c *Consumer) alreadyProcessed(msg *Message) bool {
	c.queue.mu.RLock()
	defer c.queue.mu.RUnlock()

	return msg.Seq < c.queue.offsets[c.groupID][c.topic]
}

func (c *Consumer) advanceOffset(msg *Message) {
	c.queue.mu.Lock()
	defer c.queue.mu.Unlock()

	if c.queue.offsets[c.groupID] == nil {
		c.queue.offsets[c.groupID] = make(map[string]uint64)
	}

	// Offsets track the next sequence to consume, so the very first message
	// (seq 0) is distinguishable from "nothing processed yet".
	if next := msg.Seq + 1; next > c.queue.offsets[c.groupID][c.topic] {
		c.queue.offsets[c.groupID][c.topic] = next
	}
}

func (c *Consumer) Stop(_ context.Context) error {
	c.cancel()
	c.wg.Wait()

	return nil
}
