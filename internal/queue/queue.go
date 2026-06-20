package queue

import (
	"context"

	"go.uber.org/zap"
)

type Producer interface {
	Produce(topic string, key string, value any)
	Close() error
}

type Consumer interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

type MessageHandler func(ctx context.Context, key string, value []byte) error

type ConsumerMaker struct {
	NewConsumer func(topic, groupID string, handler MessageHandler, logger *zap.SugaredLogger) (Consumer, error)
}
