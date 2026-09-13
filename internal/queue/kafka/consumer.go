package kafka

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"github.com/hexsans/hexmagnet/internal/backoff"
	"github.com/hexsans/hexmagnet/internal/queue/delivery"
	"github.com/hexsans/hexmagnet/internal/queue/permanent"
	"github.com/hexsans/hexmagnet/internal/utils"
	"go.uber.org/zap"
)

const consumerRetryDelay = 5 * time.Second

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

	deliveryMu    sync.Mutex
	deliveryCfg   delivery.Config
	attemptCounts map[string]int
}

// SetDelivery overrides the default delivery policy for this consumer.
func (c *Consumer) SetDelivery(cfg delivery.Config) {
	c.deliveryMu.Lock()
	defer c.deliveryMu.Unlock()

	c.deliveryCfg = cfg.Normalize()
}

func (c *Consumer) deliveryConfig() delivery.Config {
	c.deliveryMu.Lock()
	defer c.deliveryMu.Unlock()

	return c.deliveryCfg.Normalize()
}

// recordAttempt increments and returns the delivery attempt count for a
// message. Counts only live in memory; a process restart resets them to zero.
func (c *Consumer) recordAttempt(key string) int {
	c.deliveryMu.Lock()
	defer c.deliveryMu.Unlock()

	if c.attemptCounts == nil {
		c.attemptCounts = make(map[string]int)
	}

	c.attemptCounts[key]++

	return c.attemptCounts[key]
}

func (c *Consumer) clearAttempt(key string) {
	c.deliveryMu.Lock()
	defer c.deliveryMu.Unlock()

	delete(c.attemptCounts, key)
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
				ctx:      c.handlerCtx,
				handler:  c.handler,
				logger:   c.logger,
				consumer: c,
			}); err != nil {
				if ctx.Err() != nil {
					return
				}

				c.logger.Errorw("kafka consumer error, retrying", "topic", c.topic, "error", err)

				if (backoff.Config{Base: consumerRetryDelay, Factor: 1, Max: consumerRetryDelay}).Wait(ctx, 1) != nil {
					return
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
	ctx      context.Context
	handler  MessageHandler
	logger   *zap.SugaredLogger
	consumer *Consumer
}

func (*consumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

func (*consumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (h *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	defer utils.Recover(h.logger, "kafka consumer handler panicked")

	deliveryCfg := delivery.DefaultConfig()
	if h.consumer != nil {
		deliveryCfg = h.consumer.deliveryConfig()
	}

	for {
		select {
		case <-h.ctx.Done():
			return nil
		case msg, ok := <-claim.Messages():
			if !ok {
				return nil
			}

			err := h.handler(h.ctx, msg.Key, msg.Value)
			if err == nil {
				h.acknowledge(session, msg)
				continue
			}

			switch {
			case errors.Is(err, context.Canceled) && h.ctx.Err() != nil:
				// Shutting down: leave the message unacknowledged so it is
				// redelivered when the consumer restarts.
				h.logger.Warnw("consumer shutting down, message not acknowledged", "topic", msg.Topic)

				return nil
			case permanent.Is(err):
				h.logger.Debugw("skipping permanently failed message", "topic", msg.Topic, "error", err)
				h.acknowledge(session, msg)
			default:
				attempt := h.recordAttempt(msg)

				if attempt >= deliveryCfg.MaxAttempts {
					h.logger.Errorw("queue message exceeded delivery attempts, discarding",
						"topic", msg.Topic,
						"partition", msg.Partition,
						"offset", msg.Offset,
						"attempts", attempt,
						"error", err,
					)
					h.acknowledge(session, msg)

					continue
				}

				h.logger.Warnw("message handler failed, redelivering",
					"topic", msg.Topic,
					"partition", msg.Partition,
					"offset", msg.Offset,
					"attempt", attempt,
					"error", err,
				)

				// Honor the configured backoff before aborting the claim.
				// Sarama does not surface ConsumeClaim errors to the caller,
				// so the delay must happen here; the claim exit forces a
				// rebalance and the message is redelivered from the last
				// committed offset.
				if waitErr := deliveryCfg.Backoff.Wait(h.ctx, attempt); waitErr != nil {
					h.logger.Warnw("consumer shutting down while waiting to redeliver",
						"topic", msg.Topic,
						"partition", msg.Partition,
						"offset", msg.Offset,
					)

					return nil
				}

				return fmt.Errorf("message handler failed (attempt %d/%d): %w", attempt, deliveryCfg.MaxAttempts, err)
			}
		}
	}
}

func (h *consumerGroupHandler) acknowledge(session sarama.ConsumerGroupSession, msg *sarama.ConsumerMessage) {
	if h.consumer != nil {
		h.consumer.clearAttempt(attemptKey(msg))
	}

	session.MarkMessage(msg, "")
}

func (h *consumerGroupHandler) recordAttempt(msg *sarama.ConsumerMessage) int {
	if h.consumer == nil {
		return 1
	}

	return h.consumer.recordAttempt(attemptKey(msg))
}

func attemptKey(msg *sarama.ConsumerMessage) string {
	return fmt.Sprintf("%s/%d/%d", msg.Topic, msg.Partition, msg.Offset)
}

var _ sarama.ConsumerGroupHandler = (*consumerGroupHandler)(nil)
