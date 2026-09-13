package queue

import (
	"context"
	"sync"

	"github.com/hexsans/hexmagnet/internal/concurrency"
	"github.com/hexsans/hexmagnet/internal/queue/delivery"
	kafka2 "github.com/hexsans/hexmagnet/internal/queue/kafka"
	"github.com/hexsans/hexmagnet/internal/queue/memoryqueue"
	"go.uber.org/zap"
)

type Runtime struct {
	ActiveBackend *concurrency.AtomicValue[string]

	Producer      *concurrency.AtomicValue[Producer]
	ConsumerMaker *concurrency.AtomicValue[ConsumerMaker]
	Manager       *concurrency.AtomicValue[Manager]

	memoryQueue        *memoryqueue.MemoryQueue
	memoryManager      Manager
	memoryQueueActive  bool
	kafkaConfig        kafka2.Config
	deliveryCfg        delivery.Config
	kafkaProducerFn    func() Producer
	kafkaConsumerFn    func() ConsumerMaker
	kafkaManagerFn     func() Manager
	memoryProducer     Producer
	memoryProducerFn   func() Producer
	memoryConsumerFn   func() ConsumerMaker
	logger             *zap.SugaredLogger
	mu                 sync.Mutex
	noopKafkaFactories bool

	backendChangeMu     sync.Mutex
	backendChangeSubs   map[int]chan struct{}
	backendChangeNextID int
}

func NewRuntime(cfg Config, logger *zap.SugaredLogger) *Runtime {
	mq := memoryqueue.New(logger)

	r := &Runtime{
		ActiveBackend:     &concurrency.AtomicValue[string]{},
		Producer:          &concurrency.AtomicValue[Producer]{},
		ConsumerMaker:     &concurrency.AtomicValue[ConsumerMaker]{},
		Manager:           &concurrency.AtomicValue[Manager]{},
		memoryQueue:       mq,
		memoryManager:     &memoryManager{queue: mq},
		kafkaConfig:       cfg.Kafka,
		deliveryCfg:       cfg.deliveryConfig(),
		logger:            logger,
		backendChangeSubs: make(map[int]chan struct{}),
	}

	r.memoryProducer = &memoryProducer{mq: mq}
	r.memoryProducerFn = func() Producer { return r.memoryProducer }
	r.memoryConsumerFn = func() ConsumerMaker {
		return ConsumerMaker{
			NewConsumer: func(topic, groupID string, handler MessageHandler, l *zap.SugaredLogger) (Consumer, error) {
				return mq.NewConsumer(topic, groupID, memoryqueue.MessageHandler(handler), l), nil
			},
		}
	}

	r.rebuildKafkaClosures()
	r.applyConfig(cfg)

	return r
}

func (r *Runtime) rebuildKafkaClosures() {
	if r.noopKafkaFactories {
		r.kafkaProducerFn = func() Producer { return nil }
		r.kafkaManagerFn = func() Manager { return nil }

		return
	}

	cfg := r.kafkaConfig
	deliveryCfg := r.deliveryCfg

	r.kafkaProducerFn = func() Producer {
		prod, err := kafka2.NewProducer(cfg.Brokers, r.logger.Named("kafka"))
		if err != nil {
			r.logger.Warnw("failed to create kafka producer", "error", err)
			return nil
		}

		return prod
	}
	r.kafkaConsumerFn = func() ConsumerMaker {
		return ConsumerMaker{
			NewConsumer: func(topic, groupID string, handler MessageHandler, l *zap.SugaredLogger) (Consumer, error) {
				consumer, err := kafka2.NewConsumer(cfg.Brokers, topic, groupID, func(ctx context.Context, key, value []byte) error {
					return handler(ctx, string(key), value)
				}, l)
				if err != nil {
					return nil, err
				}

				consumer.SetDelivery(deliveryCfg)

				return consumer, nil
			},
		}
	}
	r.kafkaManagerFn = func() Manager {
		ac, err := kafka2.NewAdminClient(cfg.Brokers, r.logger.Named("queue_manager"))
		if err != nil {
			r.logger.Warnw("failed to create kafka admin client", "error", err)
			return nil
		}

		return &manager{adminClient: ac, logger: r.logger.Named("queue_manager")}
	}
}

func (r *Runtime) SwitchTo(cfg Config) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	oldBackend := r.ActiveBackend.Get()
	r.kafkaConfig = cfg.Kafka
	r.deliveryCfg = cfg.deliveryConfig()
	r.rebuildKafkaClosures()
	r.applyConfig(cfg)

	if cfg.Backend == backendMemory && oldBackend != backendMemory {
		r.memoryQueue.Reset()

		r.memoryQueueActive = true
		go r.startMemoryPersistLoop(context.Background())
	} else if cfg.Backend != backendMemory && oldBackend == backendMemory {
		_ = r.CloseMemoryQueue(context.Background())
	}

	r.notifyBackendChange()

	return nil
}

func (r *Runtime) applyConfig(cfg Config) {
	oldManager := r.Manager.Get()
	oldProducer := r.Producer.Get()

	r.memoryQueue.SetDelivery(r.deliveryCfg)

	switch cfg.Backend {
	case backendKafka:
		r.ActiveBackend.Set(backendKafka)
		r.Producer.Set(r.kafkaProducerFn())
		r.ConsumerMaker.Set(r.kafkaConsumerFn())
		r.Manager.Set(r.kafkaManagerFn())
	default:
		r.ActiveBackend.Set(backendMemory)
		r.Producer.Set(r.memoryProducerFn())
		r.ConsumerMaker.Set(r.memoryConsumerFn())
		r.Manager.Set(r.memoryManager)
	}

	if oldManager != nil {
		oldManager.Close()
	}

	if oldProducer != nil && oldProducer != r.memoryProducer {
		_ = oldProducer.Close()
	}
}

type memoryProducer struct {
	mq *memoryqueue.MemoryQueue
}

func (p *memoryProducer) Produce(topic string, key string, value any) {
	p.mq.Produce(topic, key, value)
}

func (p *memoryProducer) Close() error {
	return p.mq.Close(context.Background())
}

type dynamicProducer struct {
	r *Runtime
}

func (p *dynamicProducer) Produce(topic string, key string, value any) {
	prod := p.r.Producer.Get()
	if prod == nil {
		p.r.logger.Warnw("producer is nil, dropping message", "topic", topic)
		return
	}

	prod.Produce(topic, key, value)
}

func (p *dynamicProducer) Close() error {
	prod := p.r.Producer.Get()
	if prod == nil {
		return nil
	}

	return prod.Close()
}

func (r *Runtime) DynamicProducer() Producer {
	return &dynamicProducer{r: r}
}

func (r *Runtime) DynamicConsumerMaker() ConsumerMaker {
	return ConsumerMaker{
		NewConsumer: func(topic, groupID string, handler MessageHandler, logger *zap.SugaredLogger) (Consumer, error) {
			cm := r.ConsumerMaker.Get()
			return cm.NewConsumer(topic, groupID, handler, logger)
		},
	}
}

func (r *Runtime) DynamicManager() Manager {
	return &dynamicManager{r: r}
}

type dynamicManager struct {
	r *Runtime
}

func (m *dynamicManager) Metrics(ctx context.Context, query MetricsQuery) (MetricsResult, error) {
	mgr := m.r.Manager.Get()
	if mgr == nil {
		return MetricsResult{Buckets: []MetricsBucket{}}, nil
	}

	return mgr.Metrics(ctx, query)
}

func (m *dynamicManager) Jobs(ctx context.Context, query JobsQuery) (JobsResult, error) {
	mgr := m.r.Manager.Get()
	if mgr == nil {
		return JobsResult{Items: []Job{}, Aggregations: Aggregations{Queue: []AggQueue{}}}, nil
	}

	return mgr.Jobs(ctx, query)
}

func (m *dynamicManager) Close() error {
	mgr := m.r.Manager.Get()
	if mgr == nil {
		return nil
	}

	return mgr.Close()
}

func (r *Runtime) SubscribeBackendChanges() (<-chan struct{}, func()) {
	r.backendChangeMu.Lock()
	defer r.backendChangeMu.Unlock()

	ch := make(chan struct{}, 1)
	id := r.backendChangeNextID
	r.backendChangeNextID++
	r.backendChangeSubs[id] = ch

	return ch, func() {
		r.backendChangeMu.Lock()
		defer r.backendChangeMu.Unlock()

		delete(r.backendChangeSubs, id)
		close(ch)
	}
}

func (r *Runtime) notifyBackendChange() {
	r.backendChangeMu.Lock()
	defer r.backendChangeMu.Unlock()

	for _, ch := range r.backendChangeSubs {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

func (r *Runtime) startMemoryPersistLoop(ctx context.Context) {
	r.memoryQueue.StartPersistLoop(ctx)
}

func (r *Runtime) CloseMemoryQueue(ctx context.Context) error {
	if !r.memoryQueueActive {
		return nil
	}

	r.memoryQueueActive = false

	return r.memoryQueue.Close(ctx)
}
