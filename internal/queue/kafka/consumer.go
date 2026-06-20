package kafka

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"go.uber.org/zap"
)

type MessageHandler func(ctx context.Context, key, value []byte) error

type Consumer struct {
	client        sarama.ConsumerGroup
	topic         string
	groupID       string
	handler       MessageHandler
	logger        *zap.SugaredLogger
	wg            sync.WaitGroup
	cancel        context.CancelFunc
	handlerCtx    context.Context
	handlerCancel context.CancelFunc
	started       bool
	mu            sync.Mutex
}

func NewConsumer(brokers []string, topic string, groupID string, handler MessageHandler, logger *zap.SugaredLogger) (*Consumer, error) {
	config := sarama.NewConfig()
	config.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.NewBalanceStrategyRoundRobin()}
	config.Consumer.Offsets.Initial = sarama.OffsetNewest
	config.Consumer.MaxProcessingTime = 10 * time.Minute
	config.Metadata.AllowAutoTopicCreation = true

	client, err := sarama.NewConsumerGroup(brokers, groupID, config)
	if err != nil {
		return nil, err
	}

	return &Consumer{
		client:  client,
		topic:   topic,
		groupID: groupID,
		handler: handler,
		logger:  logger,
	}, nil
}

func (c *Consumer) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.started {
		return nil
	}

	ctx, c.cancel = context.WithCancel(ctx)
	c.handlerCtx, c.handlerCancel = context.WithCancel(ctx)
	c.started = true

	c.logger.Infow("kafka consumer started", "topic", c.topic, "group", c.groupID)

	c.wg.Go(func() {
		for {
			if err := c.client.Consume(ctx, []string{c.topic}, &consumerGroupHandler{
				ctx:     c.handlerCtx,
				handler: c.handler,
				logger:  c.logger,
			}); err != nil {
				if ctx.Err() != nil {
					return
				}

				c.logger.Errorw("kafka consumer error, retrying", "topic", c.topic, "error", err)

				select {
				case <-ctx.Done():
					return
				case <-time.After(5 * time.Second):
				}

				continue
			}

			if ctx.Err() != nil {
				return
			}
		}
	})

	return nil
}

func (c *Consumer) Stop(_ context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.started {
		return nil
	}

	c.handlerCancel()
	c.cancel()
	c.wg.Wait()
	c.started = false

	return c.client.Close()
}

type consumerGroupHandler struct {
	ctx     context.Context
	handler MessageHandler
	logger  *zap.SugaredLogger
}

func (*consumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

func (*consumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (h *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	defer func() {
		if r := recover(); r != nil {
			h.logger.Errorw("kafka consumer handler panicked",
				"panic", r,
			)
		}
	}()

	for {
		select {
		case <-h.ctx.Done():
			return nil
		case msg, ok := <-claim.Messages():
			if !ok {
				return nil
			}

			if err := h.handler(h.ctx, msg.Key, msg.Value); err != nil {
				if errors.Is(err, context.Canceled) {
					h.logger.Warnw("consumer shutting down, message skipped", "topic", msg.Topic)
				} else {
					h.logger.Errorw("error handling message", "topic", msg.Topic, "error", err)
				}
			}

			session.MarkMessage(msg, "")
		}
	}
}

var _ sarama.ConsumerGroupHandler = (*consumerGroupHandler)(nil)
