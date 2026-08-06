package queue

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hexsans/hexmagnet/internal/queue/kafka"
	"github.com/hexsans/hexmagnet/internal/queue/memoryqueue"
	"github.com/hexsans/hexmagnet/internal/utils"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func newMemoryManager(mq *memoryqueue.MemoryQueue) utils.Lazy[Manager] {
	return utils.NewLazy(func() (Manager, error) { return &memoryManager{queue: mq}, nil })
}

type mockAdminClient struct {
	listConsumerGroupsFn    func() ([]kafka.ConsumerGroupSummary, error)
	describeConsumerGroupFn func(string) (*kafka.GroupDetail, error)
	listTopicsFn            func() ([]kafka.TopicInfo, error)
	closeFn                 func() error
}

func (m *mockAdminClient) ListConsumerGroups() ([]kafka.ConsumerGroupSummary, error) {
	return m.listConsumerGroupsFn()
}

func (m *mockAdminClient) DescribeConsumerGroup(groupID string) (*kafka.GroupDetail, error) {
	return m.describeConsumerGroupFn(groupID)
}

func (m *mockAdminClient) ListTopics() ([]kafka.TopicInfo, error) {
	if m.listTopicsFn != nil {
		return m.listTopicsFn()
	}

	return nil, nil
}

func (m *mockAdminClient) Close() error {
	if m.closeFn != nil {
		return m.closeFn()
	}

	return nil
}

func TestDefaultBackend(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "memory", defaultBackend())
}

func TestNewDefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := NewDefaultConfig()
	assert.Equal(t, "memory", cfg.Backend)
	assert.NotZero(t, cfg.Kafka)
}

func TestNewModule_Memory(t *testing.T) {
	t.Parallel()

	mod := NewModule(Config{Backend: "memory"})
	assert.NotNil(t, mod)
}

func TestNewModule_Kafka(t *testing.T) {
	t.Parallel()

	mod := NewModule(Config{Backend: "kafka"})
	assert.NotNil(t, mod)
}

func TestMapGroupState(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "processed", mapGroupState("Stable"))
	assert.Equal(t, "pending", mapGroupState("Empty"))
	assert.Equal(t, "failed", mapGroupState("Dead"))
	assert.Equal(t, "retry", mapGroupState("PreparingRebalance"))
	assert.Equal(t, "retry", mapGroupState("CompletingRebalance"))
	assert.Equal(t, "pending", mapGroupState("Unknown"))
	assert.Equal(t, "pending", mapGroupState(""))
}

func TestNewManager(t *testing.T) {
	t.Parallel()

	mockAC := &mockAdminClient{
		listConsumerGroupsFn: func() ([]kafka.ConsumerGroupSummary, error) {
			return nil, nil
		},
	}
	lazyAC := utils.NewLazy(func() (kafka.AdminClient, error) { return mockAC, nil })
	mgr := newTestManager(lazyAC)
	assert.NotNil(t, mgr)
}

func TestManagerMetrics_ListError(t *testing.T) {
	t.Parallel()

	mockAC := &mockAdminClient{
		listConsumerGroupsFn: func() ([]kafka.ConsumerGroupSummary, error) {
			return nil, assert.AnError
		},
	}
	lazyAC := utils.NewLazy(func() (kafka.AdminClient, error) { return mockAC, nil })
	mgr := newTestManager(lazyAC)

	res, err := mgr.Metrics(context.Background(), MetricsQuery{})
	require.NoError(t, err)
	assert.Empty(t, res.Buckets)
}

func TestManagerMetrics_Success(t *testing.T) {
	t.Parallel()

	mockAC := &mockAdminClient{
		listConsumerGroupsFn: func() ([]kafka.ConsumerGroupSummary, error) {
			return []kafka.ConsumerGroupSummary{
				{GroupID: "group-a", State: "Stable"},
				{GroupID: "group-b", State: "Empty"},
			}, nil
		},
		describeConsumerGroupFn: func(groupID string) (*kafka.GroupDetail, error) {
			return &kafka.GroupDetail{
				GroupID: groupID,
				State:   "Stable",
				Topics: []kafka.TopicDetail{
					{
						Topic: "topic-1",
						Partitions: []kafka.PartitionOffset{
							{Partition: 0, Lag: 5},
						},
					},
				},
			}, nil
		},
	}
	lazyAC := utils.NewLazy(func() (kafka.AdminClient, error) { return mockAC, nil })
	mgr := newTestManager(lazyAC)

	res, err := mgr.Metrics(context.Background(), MetricsQuery{BucketDuration: time.Hour})
	require.NoError(t, err)
	require.Len(t, res.Buckets, 2)
	assert.Equal(t, "group-a / topic-1", res.Buckets[0].Queue)
	assert.Equal(t, "processed", res.Buckets[0].Status)
	assert.Equal(t, uint(5), res.Buckets[0].Count)
	assert.Equal(t, "group-b / topic-1", res.Buckets[1].Queue)
	assert.Equal(t, "pending", res.Buckets[1].Status)
}

func TestManagerMetrics_StatusFilter(t *testing.T) {
	t.Parallel()

	mockAC := &mockAdminClient{
		listConsumerGroupsFn: func() ([]kafka.ConsumerGroupSummary, error) {
			return []kafka.ConsumerGroupSummary{
				{GroupID: "g1", State: "Stable"},
				{GroupID: "g2", State: "Empty"},
			}, nil
		},
		describeConsumerGroupFn: func(groupID string) (*kafka.GroupDetail, error) {
			return &kafka.GroupDetail{
				GroupID: groupID,
				Topics: []kafka.TopicDetail{
					{Topic: "t1", Partitions: []kafka.PartitionOffset{{Partition: 0}}},
				},
			}, nil
		},
	}
	lazyAC := utils.NewLazy(func() (kafka.AdminClient, error) { return mockAC, nil })
	mgr := newTestManager(lazyAC)

	res, err := mgr.Metrics(context.Background(), MetricsQuery{
		BucketDuration: time.Hour,
		Statuses:       []string{"processed"},
	})
	require.NoError(t, err)
	require.Len(t, res.Buckets, 1)
	assert.Equal(t, "processed", res.Buckets[0].Status)
}

func TestManagerMetrics_CancelledContext(t *testing.T) {
	t.Parallel()

	mockAC := &mockAdminClient{
		listConsumerGroupsFn: func() ([]kafka.ConsumerGroupSummary, error) {
			return []kafka.ConsumerGroupSummary{{GroupID: "g1", State: "Stable"}}, nil
		},
		describeConsumerGroupFn: func(groupID string) (*kafka.GroupDetail, error) {
			return &kafka.GroupDetail{
				GroupID: groupID,
				Topics:  []kafka.TopicDetail{{Topic: "t1"}},
			}, nil
		},
	}
	lazyAC := utils.NewLazy(func() (kafka.AdminClient, error) { return mockAC, nil })
	mgr := newTestManager(lazyAC)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := mgr.Metrics(ctx, MetricsQuery{BucketDuration: time.Hour})
	assert.ErrorIs(t, err, context.Canceled)
}

func TestManagerJobs_ListError(t *testing.T) {
	t.Parallel()

	mockAC := &mockAdminClient{
		listConsumerGroupsFn: func() ([]kafka.ConsumerGroupSummary, error) {
			return nil, assert.AnError
		},
	}
	lazyAC := utils.NewLazy(func() (kafka.AdminClient, error) { return mockAC, nil })
	mgr := newTestManager(lazyAC)

	res, err := mgr.Jobs(context.Background(), JobsQuery{Offset: 0})
	require.NoError(t, err)
	assert.False(t, res.HasNextPage)
	assert.Empty(t, res.Items)
}

func TestManagerJobs_Success(t *testing.T) {
	t.Parallel()

	mockAC := &mockAdminClient{
		listConsumerGroupsFn: func() ([]kafka.ConsumerGroupSummary, error) {
			return []kafka.ConsumerGroupSummary{{GroupID: "g1", State: "Stable"}}, nil
		},
		describeConsumerGroupFn: func(groupID string) (*kafka.GroupDetail, error) {
			return &kafka.GroupDetail{
				GroupID: groupID,
				Topics: []kafka.TopicDetail{
					{Topic: "t1", Partitions: []kafka.PartitionOffset{{Partition: 0, Lag: 3}}},
				},
			}, nil
		},
	}
	lazyAC := utils.NewLazy(func() (kafka.AdminClient, error) { return mockAC, nil })
	mgr := newTestManager(lazyAC)

	res, err := mgr.Jobs(context.Background(), JobsQuery{Offset: 0})
	require.NoError(t, err)
	require.Len(t, res.Items, 1)
	assert.Equal(t, "g1/t1", res.Items[0].ID)
	assert.Equal(t, "t1", res.Items[0].Queue)
	assert.Contains(t, res.Items[0].Payload, `"lag":3`)
	assert.Equal(t, 1, res.TotalCount)
}

func TestManagerJobs_QueueFilter(t *testing.T) {
	t.Parallel()

	mockAC := &mockAdminClient{
		listConsumerGroupsFn: func() ([]kafka.ConsumerGroupSummary, error) {
			return []kafka.ConsumerGroupSummary{{GroupID: "g1", State: "Stable"}}, nil
		},
		describeConsumerGroupFn: func(groupID string) (*kafka.GroupDetail, error) {
			return &kafka.GroupDetail{
				GroupID: groupID,
				Topics: []kafka.TopicDetail{
					{Topic: "t1", Partitions: []kafka.PartitionOffset{{Partition: 0, Lag: 1}}},
					{Topic: "t2", Partitions: []kafka.PartitionOffset{{Partition: 0, Lag: 2}}},
				},
			}, nil
		},
	}
	lazyAC := utils.NewLazy(func() (kafka.AdminClient, error) { return mockAC, nil })
	mgr := newTestManager(lazyAC)

	res, err := mgr.Jobs(context.Background(), JobsQuery{
		Queues: &FacetFilter{Values: []string{"t1"}, Logic: "or"},
	})
	require.NoError(t, err)
	require.Len(t, res.Items, 1)
	assert.Equal(t, "t1", res.Items[0].Queue)
}

func TestManagerJobs_Pagination(t *testing.T) {
	t.Parallel()

	mockAC := &mockAdminClient{
		listConsumerGroupsFn: func() ([]kafka.ConsumerGroupSummary, error) {
			return []kafka.ConsumerGroupSummary{{GroupID: "g1", State: "Stable"}}, nil
		},
		describeConsumerGroupFn: func(groupID string) (*kafka.GroupDetail, error) {
			return &kafka.GroupDetail{
				GroupID: groupID,
				Topics: []kafka.TopicDetail{
					{Topic: "t1", Partitions: []kafka.PartitionOffset{{Partition: 0}}},
					{Topic: "t2", Partitions: []kafka.PartitionOffset{{Partition: 0}}},
				},
			}, nil
		},
	}
	lazyAC := utils.NewLazy(func() (kafka.AdminClient, error) { return mockAC, nil })
	mgr := newTestManager(lazyAC)

	res, err := mgr.Jobs(context.Background(), JobsQuery{Offset: 0, Limit: 1})
	require.NoError(t, err)
	assert.Len(t, res.Items, 1)
	assert.True(t, res.HasNextPage)
	assert.Equal(t, 2, res.TotalCount)
}

func TestManagerJobs_CancelledContext(t *testing.T) {
	t.Parallel()

	mockAC := &mockAdminClient{
		listConsumerGroupsFn: func() ([]kafka.ConsumerGroupSummary, error) {
			return []kafka.ConsumerGroupSummary{{GroupID: "g1", State: "Stable"}}, nil
		},
		describeConsumerGroupFn: func(groupID string) (*kafka.GroupDetail, error) {
			return &kafka.GroupDetail{
				GroupID: groupID,
				Topics:  []kafka.TopicDetail{{Topic: "t1"}},
			}, nil
		},
	}
	lazyAC := utils.NewLazy(func() (kafka.AdminClient, error) { return mockAC, nil })
	mgr := newTestManager(lazyAC)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := mgr.Jobs(ctx, JobsQuery{Offset: 0})
	assert.ErrorIs(t, err, context.Canceled)
}

func TestMemoryManager_New(t *testing.T) {
	t.Parallel()

	mq := memoryqueue.New(zap.NewNop().Sugar())
	lazyMgr := newMemoryManager(mq)
	mgr, err := lazyMgr.Get()
	require.NoError(t, err)
	assert.NotNil(t, mgr)
}

func TestMemoryManagerMetrics(t *testing.T) {
	t.Parallel()

	mq := memoryqueue.New(zap.NewNop().Sugar())
	lazyMgr := newMemoryManager(mq)
	mgr, err := lazyMgr.Get()
	require.NoError(t, err)

	res, err := mgr.Metrics(context.Background(), MetricsQuery{})
	require.NoError(t, err)
	assert.Empty(t, res.Buckets)
}

func TestMemoryManagerJobs_Empty(t *testing.T) {
	t.Parallel()

	mq := memoryqueue.New(zap.NewNop().Sugar())
	lazyMgr := newMemoryManager(mq)
	mgr, err := lazyMgr.Get()
	require.NoError(t, err)

	res, err := mgr.Jobs(context.Background(), JobsQuery{Offset: 0})
	require.NoError(t, err)
	assert.Empty(t, res.Items)
	assert.Equal(t, 0, res.TotalCount)
	assert.False(t, res.HasNextPage)
}

func TestMemoryManagerJobs_WithMessages(t *testing.T) {
	t.Parallel()

	mq := memoryqueue.New(zap.NewNop().Sugar())
	mq.NewConsumer("topic-a", "group-1", nil, zap.NewNop().Sugar())
	mq.NewConsumer("topic-b", "group-1", nil, zap.NewNop().Sugar())
	mq.Produce("topic-a", "k1", "v1")
	mq.Produce("topic-a", "k2", "v2")
	mq.Produce("topic-b", "k3", "v3")

	lazyMgr := newMemoryManager(mq)
	mgr, err := lazyMgr.Get()
	require.NoError(t, err)

	res, err := mgr.Jobs(context.Background(), JobsQuery{Offset: 0})
	require.NoError(t, err)
	require.Len(t, res.Items, 2)
	assert.Equal(t, "group-1/topic-a", res.Items[0].ID)
	assert.Equal(t, "group-1/topic-b", res.Items[1].ID)
	assert.Equal(t, 2, res.TotalCount)
	require.Len(t, res.Aggregations.Queue, 2)
	assert.Equal(t, "topic-a", res.Aggregations.Queue[0].Value)
	assert.Equal(t, 1, res.Aggregations.Queue[0].Count)
	assert.Equal(t, "topic-b", res.Aggregations.Queue[1].Value)
	assert.Equal(t, 1, res.Aggregations.Queue[1].Count)
}

func TestMemoryManagerJobs_QueueFilter(t *testing.T) {
	t.Parallel()

	mq := memoryqueue.New(zap.NewNop().Sugar())
	mq.NewConsumer("topic-a", "g1", nil, zap.NewNop().Sugar())
	mq.NewConsumer("topic-b", "g1", nil, zap.NewNop().Sugar())
	mq.Produce("topic-a", "k1", "v1")
	mq.Produce("topic-b", "k2", "v2")

	lazyMgr := newMemoryManager(mq)
	mgr, err := lazyMgr.Get()
	require.NoError(t, err)

	res, err := mgr.Jobs(context.Background(), JobsQuery{
		Queues: &FacetFilter{Values: []string{"topic-a"}, Logic: "or"},
	})
	require.NoError(t, err)
	require.Len(t, res.Items, 1)
	assert.Equal(t, "topic-a", res.Items[0].Queue)
}

func TestMemoryManagerJobs_Pagination(t *testing.T) {
	t.Parallel()

	mq := memoryqueue.New(zap.NewNop().Sugar())
	mq.NewConsumer("t1", "g1", nil, zap.NewNop().Sugar())
	mq.NewConsumer("t2", "g1", nil, zap.NewNop().Sugar())
	mq.NewConsumer("t3", "g1", nil, zap.NewNop().Sugar())
	mq.Produce("t1", "k", "v")
	mq.Produce("t2", "k", "v")
	mq.Produce("t3", "k", "v")

	lazyMgr := newMemoryManager(mq)
	mgr, err := lazyMgr.Get()
	require.NoError(t, err)

	res, err := mgr.Jobs(context.Background(), JobsQuery{Offset: 0, Limit: 2})
	require.NoError(t, err)
	require.Len(t, res.Items, 2)
	assert.True(t, res.HasNextPage)
	assert.Equal(t, 3, res.TotalCount)
}

func TestNewModule_Memory_ProvidesComponents(t *testing.T) {
	t.Parallel()

	app := fx.New(
		fx.Provide(
			func() *zap.SugaredLogger { return zap.NewNop().Sugar() },
			func() utils.Lazy[*pgxpool.Pool] {
				return utils.NewLazy(func() (*pgxpool.Pool, error) {
					return nil, errors.New("no pool in test")
				})
			},
		),
		NewModule(Config{Backend: "memory"}),
		fx.Invoke(func(Producer, ConsumerMaker, utils.Lazy[Manager]) {}),
	)
	require.NoError(t, app.Err())
}

func TestMemoryManagerJobs_CancelledContext(t *testing.T) {
	t.Parallel()

	mq := memoryqueue.New(zap.NewNop().Sugar())
	mq.NewConsumer("t1", "g1", nil, zap.NewNop().Sugar())
	mq.Produce("t1", "k", "v")

	lazyMgr := newMemoryManager(mq)
	mgr, err := lazyMgr.Get()
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = mgr.Jobs(ctx, JobsQuery{Offset: 0})
	assert.ErrorIs(t, err, context.Canceled)
}

func newTestManager(lazyAC utils.Lazy[kafka.AdminClient]) Manager {
	ac, err := lazyAC.Get()
	if err != nil {
		panic(err)
	}

	return &manager{adminClient: ac, logger: zap.NewNop().Sugar()}
}
