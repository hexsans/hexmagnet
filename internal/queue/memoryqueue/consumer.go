package memoryqueue

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/hexsans/hexmagnet/internal/queue/permanent"
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
		defer func() {
			if r := recover(); r != nil {
				c.logger.Errorw("memory queue consumer panicked",
					"topic", c.topic, "group", c.groupID, "panic", r,
				)
			}
		}()

		c.run(ctx)
	}()

	return nil
}

func (c *Consumer) run(ctx context.Context) {
	defer c.wg.Done()

	c.replay(ctx)

	for {
		select {
		case msg, ok := <-c.ch:
			if !ok {
				return
			}

			c.process(ctx, msg)
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
		if msg != nil && msg.Seq > offset {
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

func (c *Consumer) process(ctx context.Context, msg *Message) {
	if err := c.handler(ctx, msg.Key, []byte(msg.Value)); err != nil {
		if permanent.Is(err) {
			c.logger.Debugw("skipping permanently failed message",
				"topic", c.topic, "group", c.groupID, "seq", msg.Seq, "error", err)

			c.advanceOffset(msg)

			return
		}

		c.logger.Debugw("message handler failed",
			"topic", c.topic, "group", c.groupID, "seq", msg.Seq, "error", err)

		return
	}

	c.advanceOffset(msg)
}

func (c *Consumer) advanceOffset(msg *Message) {
	c.queue.mu.Lock()
	defer c.queue.mu.Unlock()

	if c.queue.offsets[c.groupID] == nil {
		c.queue.offsets[c.groupID] = make(map[string]uint64)
	}

	if msg.Seq > c.queue.offsets[c.groupID][c.topic] {
		c.queue.offsets[c.groupID][c.topic] = msg.Seq
	}
}

func (c *Consumer) Stop(_ context.Context) error {
	c.cancel()
	c.wg.Wait()

	return nil
}
