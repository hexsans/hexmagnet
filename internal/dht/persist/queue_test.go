package persist

import (
	"context"
	"testing"

	"github.com/hexsans/hexmagnet/internal/queue"
	"github.com/hexsans/hexmagnet/internal/testutil"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type mockProducer struct {
	queue.Producer
	producedTopic string
	producedKey   string
	producedValue any
}

func (m *mockProducer) Produce(topic string, key string, value any) {
	m.producedTopic = topic
	m.producedKey = key
	m.producedValue = value
}

func (*mockProducer) Close() error { return nil }

type mockConsumer struct {
	queue.Consumer
	started bool
	stopped bool
}

func (m *mockConsumer) Start(_ context.Context) error {
	m.started = true
	return nil
}

func (m *mockConsumer) Stop(_ context.Context) error {
	m.stopped = true
	return nil
}

func TestNewQueueConsumer(t *testing.T) {
	t.Parallel()

	handler := New(testutil.NewTestLogger())
	result := NewQueueConsumer(QueueConsumerParams{
		Handler:  handler,
		Producer: &mockProducer{},
		ConsumerMaker: queue.ConsumerMaker{
			NewConsumer: func(_, _ string, _ queue.MessageHandler, _ *zap.SugaredLogger) (queue.Consumer, error) {
				return &mockConsumer{}, nil
			},
		},
		Logger: testutil.NewTestLogger(),
	})

	assert.NotNil(t, result.Worker)
	assert.Equal(t, "dht_persist_consumer", result.Worker.Key())
}

func TestNewQueueConsumer_NilProducer(t *testing.T) {
	t.Parallel()

	handler := New(testutil.NewTestLogger())
	result := NewQueueConsumer(QueueConsumerParams{
		Handler:  handler,
		Producer: nil,
		ConsumerMaker: queue.ConsumerMaker{
			NewConsumer: func(_, _ string, _ queue.MessageHandler, _ *zap.SugaredLogger) (queue.Consumer, error) {
				return &mockConsumer{}, nil
			},
		},
		Logger: testutil.NewTestLogger(),
	})

	assert.NotNil(t, result.Worker)
	assert.Equal(t, "dht_persist_consumer", result.Worker.Key())
}
