package queue

import (
	"context"
	"testing"

	kafka2 "github.com/hexsans/hexmagnet/internal/queue/kafka"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newTestRuntime(cfg Config, logger *zap.SugaredLogger) *Runtime {
	r := NewRuntime(Config{Backend: "memory"}, logger)

	r.noopKafkaFactories = true
	if cfg.Backend != "memory" {
		_ = r.SwitchTo(cfg)
	}

	return r
}

func TestNewRuntime_Memory(t *testing.T) {
	t.Parallel()

	r := NewRuntime(Config{Backend: "memory"}, zap.NewNop().Sugar())

	assert.Equal(t, "memory", r.ActiveBackend.Get())
	assert.NotNil(t, r.Producer.Get(), "memory producer should be set")
	assert.NotNil(t, r.ConsumerMaker.Get(), "consumer maker should be set")
	assert.NotNil(t, r.Manager.Get(), "manager should be set (memory manager)")
}

func TestNewRuntime_Kafka(t *testing.T) {
	t.Parallel()

	r := newTestRuntime(Config{
		Backend: "kafka",
		Kafka: kafka2.Config{
			Brokers: []string{""},
		},
	}, zap.NewNop().Sugar())

	assert.Equal(t, "kafka", r.ActiveBackend.Get())
	assert.Nil(t, r.Producer.Get(), "kafka producer should be nil (no broker)")
}

func TestNewRuntime_DefaultIsMemory(t *testing.T) {
	t.Parallel()

	r := NewRuntime(Config{Backend: "unknown"}, zap.NewNop().Sugar())

	assert.Equal(t, "memory", r.ActiveBackend.Get())
	assert.NotNil(t, r.Producer.Get())
}

func TestRuntime_SwitchTo_SameMemory(t *testing.T) {
	t.Parallel()

	r := NewRuntime(Config{Backend: "memory"}, zap.NewNop().Sugar())

	err := r.SwitchTo(Config{Backend: "memory"})
	require.NoError(t, err)

	assert.Equal(t, "memory", r.ActiveBackend.Get())
	assert.NotNil(t, r.Producer.Get())
	assert.NotNil(t, r.Manager.Get())
}

func TestRuntime_SwitchTo_MemoryToKafka(t *testing.T) {
	t.Parallel()

	r := newTestRuntime(Config{Backend: "memory"}, zap.NewNop().Sugar())

	err := r.SwitchTo(Config{
		Backend: "kafka",
		Kafka: kafka2.Config{
			Brokers: []string{""},
		},
	})
	require.NoError(t, err)

	assert.Equal(t, "kafka", r.ActiveBackend.Get())
}

func TestRuntime_SwitchTo_KafkaToMemory(t *testing.T) {
	t.Parallel()

	r := newTestRuntime(Config{
		Backend: "kafka",
		Kafka:   kafka2.Config{Brokers: []string{""}},
	}, zap.NewNop().Sugar())

	err := r.SwitchTo(Config{Backend: "memory"})
	require.NoError(t, err)

	assert.Equal(t, "memory", r.ActiveBackend.Get())
	assert.NotNil(t, r.Producer.Get())
	assert.NotNil(t, r.Manager.Get())
}

func TestRuntime_Producer_CanProduce(t *testing.T) {
	t.Parallel()

	r := NewRuntime(Config{Backend: "memory"}, zap.NewNop().Sugar())
	prod := r.Producer.Get()
	require.NotNil(t, prod)

	prod.Produce("test-topic", "key", "value")
}

func TestRuntime_ConsumerMaker_CanCreate(t *testing.T) {
	t.Parallel()

	r := NewRuntime(Config{Backend: "memory"}, zap.NewNop().Sugar())
	cm := r.ConsumerMaker.Get()
	require.NotNil(t, cm)

	consumer, err := cm.NewConsumer("t", "g", func(_ context.Context, _ string, _ []byte) error {
		return nil
	}, zap.NewNop().Sugar())
	require.NoError(t, err)
	require.NotNil(t, consumer)
}

func TestRuntime_SwitchTo_KafkaBrokersUpdated(t *testing.T) {
	t.Parallel()

	r := newTestRuntime(Config{
		Backend: "memory",
	}, zap.NewNop().Sugar())

	require.NoError(t, r.SwitchTo(Config{
		Backend: "kafka",
		Kafka:   kafka2.Config{Brokers: []string{"old:9092"}},
	}))

	assert.Equal(t, []string{"old:9092"}, r.kafkaConfig.Brokers)

	require.NoError(t, r.SwitchTo(Config{
		Backend: "kafka",
		Kafka:   kafka2.Config{Brokers: []string{"new:9092"}},
	}))

	assert.Equal(t, []string{"new:9092"}, r.kafkaConfig.Brokers,
		"kafkaConfig should reflect new brokers after SwitchTo")
}

func TestRuntime_SwitchTo_KafkaFromMemoryUsesSwitchConfig(t *testing.T) {
	t.Parallel()

	r := newTestRuntime(Config{
		Backend: "memory",
	}, zap.NewNop().Sugar())

	err := r.SwitchTo(Config{
		Backend: "kafka",
		Kafka:   kafka2.Config{Brokers: []string{"switched:9092"}},
	})
	require.NoError(t, err)

	assert.Equal(t, "kafka", r.ActiveBackend.Get())
	assert.Equal(t, []string{"switched:9092"}, r.kafkaConfig.Brokers,
		"kafkaConfig should use SwitchTo brokers, not NewRuntime brokers")
}

func TestRuntime_SwitchTo_KafkaResetToMemory(t *testing.T) {
	t.Parallel()

	r := newTestRuntime(Config{
		Backend: "kafka",
		Kafka:   kafka2.Config{Brokers: []string{""}},
	}, zap.NewNop().Sugar())

	assert.Nil(t, r.Producer.Get(), "kafka producer should be nil (empty broker)")

	err := r.SwitchTo(Config{Backend: "memory"})
	require.NoError(t, err)

	assert.Equal(t, "memory", r.ActiveBackend.Get())
	assert.NotNil(t, r.Producer.Get(), "memory producer should be set after switch")
	assert.NotNil(t, r.Manager.Get(), "memory manager should be set after switch")
}

func TestRuntime_DynamicProducer_DelegatesProduce(t *testing.T) {
	t.Parallel()

	r := NewRuntime(Config{Backend: "memory"}, zap.NewNop().Sugar())
	dp := r.DynamicProducer()

	dp.Produce("test-topic", "key", "value")
}

func TestRuntime_DynamicProducer_DelegatesClose(t *testing.T) {
	t.Parallel()

	r := NewRuntime(Config{Backend: "memory"}, zap.NewNop().Sugar())
	dp := r.DynamicProducer()

	err := dp.Close()
	require.NoError(t, err)
}

func TestRuntime_DynamicConsumerMaker_Delegates(t *testing.T) {
	t.Parallel()

	r := NewRuntime(Config{Backend: "memory"}, zap.NewNop().Sugar())
	dcm := r.DynamicConsumerMaker()

	consumer, err := dcm.NewConsumer("t", "g", func(_ context.Context, _ string, _ []byte) error {
		return nil
	}, zap.NewNop().Sugar())
	require.NoError(t, err)
	require.NotNil(t, consumer)
}

func TestRuntime_DynamicProducer_AfterSwitch(t *testing.T) {
	t.Parallel()

	r := newTestRuntime(Config{Backend: "memory"}, zap.NewNop().Sugar())
	dp := r.DynamicProducer()

	err := r.SwitchTo(Config{
		Backend: "kafka",
		Kafka:   kafka2.Config{Brokers: []string{""}},
	})
	require.NoError(t, err)

	dp.Produce("test", "k", "v")
}

func TestRuntime_DynamicManager_AfterSwitch(t *testing.T) {
	t.Parallel()

	r := NewRuntime(Config{Backend: "memory"}, zap.NewNop().Sugar())
	dm := r.DynamicManager()

	metrics, err := dm.Metrics(context.Background(), MetricsQuery{})
	require.NoError(t, err)
	assert.NotNil(t, metrics)

	jobs, err := dm.Jobs(context.Background(), JobsQuery{})
	require.NoError(t, err)
	assert.NotNil(t, jobs)
}
