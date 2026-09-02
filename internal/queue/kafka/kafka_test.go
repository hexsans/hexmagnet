package kafka

import (
	"context"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()
	assert.Equal(t, []string{"localhost:9092"}, cfg.Brokers)
}

func TestTopicConstants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "dht.discovered_hashes", TopicDiscoveredHashes)
	assert.Equal(t, "dht.get_peers", TopicGetPeers)
	assert.Equal(t, "dht.scrape", TopicScrape)
	assert.Equal(t, "dht.metainfo", TopicMetaInfo)
	assert.Equal(t, "dht.scrape_result", TopicScrapeResult)
	assert.Equal(t, "torrents.process", TopicProcessTorrent)
	assert.Equal(t, "torrents.enriched", TopicEnriched)
}

type mockAsyncProducer struct {
	mock.Mock
}

func (m *mockAsyncProducer) AsyncClose()  { m.Called() }
func (m *mockAsyncProducer) Close() error { return m.Called().Error(0) }
func (m *mockAsyncProducer) Input() chan<- *sarama.ProducerMessage {
	return m.Called().Get(0).(chan *sarama.ProducerMessage)
}

func (m *mockAsyncProducer) Successes() <-chan *sarama.ProducerMessage {
	return m.Called().Get(0).(chan *sarama.ProducerMessage)
}

func (m *mockAsyncProducer) Errors() <-chan *sarama.ProducerError {
	return m.Called().Get(0).(chan *sarama.ProducerError)
}
func (m *mockAsyncProducer) IsTransactional() bool { return m.Called().Bool(0) }
func (m *mockAsyncProducer) TxnStatus() sarama.ProducerTxnStatusFlag {
	return m.Called().Get(0).(sarama.ProducerTxnStatusFlag)
}
func (m *mockAsyncProducer) BeginTxn() error  { return m.Called().Error(0) }
func (m *mockAsyncProducer) CommitTxn() error { return m.Called().Error(0) }
func (m *mockAsyncProducer) AbortTxn() error  { return m.Called().Error(0) }
func (m *mockAsyncProducer) AddOffsetsToTxn(offsets map[string][]*sarama.PartitionOffsetMetadata, groupID string) error {
	return m.Called(offsets, groupID).Error(0)
}

func (m *mockAsyncProducer) AddOffsetsToTxnWithGroupMetadata(
	offsets map[string][]*sarama.PartitionOffsetMetadata,
	groupMetadata *sarama.ConsumerGroupMetadata,
) error {
	return m.Called(offsets, groupMetadata).Error(0)
}

func (m *mockAsyncProducer) AddMessageToTxn(msg *sarama.ConsumerMessage, groupID string, metadata *string) error {
	return m.Called(msg, groupID, metadata).Error(0)
}

func (m *mockAsyncProducer) AddMessageToTxnWithGroupMetadata(
	msg *sarama.ConsumerMessage,
	groupMetadata *sarama.ConsumerGroupMetadata,
	metadata *string,
) error {
	return m.Called(msg, groupMetadata, metadata).Error(0)
}

func TestProducer_Produce(t *testing.T) {
	t.Parallel()

	input := make(chan *sarama.ProducerMessage, 1)

	mockProd := new(mockAsyncProducer)
	mockProd.On("Input").Return(input)

	p := &Producer{
		producer: mockProd,
		logger:   zap.NewNop().Sugar(),
	}

	p.Produce("test-topic", "test-key", map[string]string{"hello": "world"})

	select {
	case msg := <-input:
		assert.Equal(t, "test-topic", msg.Topic)
		key, _ := msg.Key.Encode()
		assert.Equal(t, []byte("test-key"), key)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for message")
	}
}

func TestProducer_Produce_MarshalError(t *testing.T) {
	t.Parallel()

	mockProd := new(mockAsyncProducer)
	mockProd.On("Input").Return(make(chan *sarama.ProducerMessage))

	p := &Producer{
		producer: mockProd,
		logger:   zap.NewNop().Sugar(),
	}

	p.Produce("test-topic", "key", make(chan int))
}

func TestProducer_Close(t *testing.T) {
	t.Parallel()

	mockProd := new(mockAsyncProducer)
	mockProd.On("Close").Return(nil)

	p := &Producer{
		producer: mockProd,
	}

	err := p.Close()
	require.NoError(t, err)
	mockProd.AssertExpectations(t)
}

type mockConsumerGroupSession struct {
	mock.Mock
}

func (m *mockConsumerGroupSession) Claims() map[string][]int32 {
	return m.Called().Get(0).(map[string][]int32)
}
func (m *mockConsumerGroupSession) MemberID() string    { return m.Called().String(0) }
func (m *mockConsumerGroupSession) GenerationID() int32 { return m.Called().Get(0).(int32) }
func (m *mockConsumerGroupSession) GroupID() string     { return m.Called().String(0) }
func (m *mockConsumerGroupSession) Commit()             { m.Called() }
func (m *mockConsumerGroupSession) MarkMessage(msg *sarama.ConsumerMessage, metadata string) {
	m.Called(msg, metadata)
}

func (m *mockConsumerGroupSession) MarkOffset(topic string, partition int32, offset int64, metadata string) {
	m.Called(topic, partition, offset, metadata)
}

func (m *mockConsumerGroupSession) ResetOffset(topic string, partition int32, offset int64, metadata string) {
	m.Called(topic, partition, offset, metadata)
}

func (m *mockConsumerGroupSession) Context() context.Context {
	return m.Called().Get(0).(context.Context)
}

type mockConsumerGroupClaim struct {
	mock.Mock
	messages chan *sarama.ConsumerMessage
}

func (m *mockConsumerGroupClaim) Topic() string              { return m.Called().String(0) }
func (m *mockConsumerGroupClaim) Partition() int32           { return m.Called().Get(0).(int32) }
func (m *mockConsumerGroupClaim) InitialOffset() int64       { return m.Called().Get(0).(int64) }
func (m *mockConsumerGroupClaim) HighWaterMarkOffset() int64 { return m.Called().Get(0).(int64) }
func (m *mockConsumerGroupClaim) Messages() <-chan *sarama.ConsumerMessage {
	return m.messages
}

func TestConsumerGroupHandler_ConsumeClaim(t *testing.T) {
	t.Parallel()

	mockSession := new(mockConsumerGroupSession)
	claim := &mockConsumerGroupClaim{
		messages: make(chan *sarama.ConsumerMessage, 1),
	}

	var handled bool

	h := &consumerGroupHandler{
		ctx: context.Background(),
		handler: func(_ context.Context, key, value []byte) error {
			handled = true

			assert.Equal(t, []byte("key"), key)
			assert.Equal(t, []byte("value"), value)

			return nil
		},
		logger: zap.NewNop().Sugar(),
	}

	mockSession.On("MarkMessage", mock.Anything, "").Return()

	claim.messages <- &sarama.ConsumerMessage{Key: []byte("key"), Value: []byte("value")}

	close(claim.messages)

	err := h.ConsumeClaim(mockSession, claim)
	require.NoError(t, err)
	assert.True(t, handled)
}

func TestConsumerGroupHandler_ConsumeClaim_HandlerError(t *testing.T) {
	t.Parallel()

	mockSession := new(mockConsumerGroupSession)
	claim := &mockConsumerGroupClaim{
		messages: make(chan *sarama.ConsumerMessage, 1),
	}

	h := &consumerGroupHandler{
		ctx: context.Background(),
		handler: func(_ context.Context, _, _ []byte) error {
			return assert.AnError
		},
		logger: zap.NewNop().Sugar(),
	}

	mockSession.On("MarkMessage", mock.Anything, "").Return()

	claim.messages <- &sarama.ConsumerMessage{Key: nil, Value: nil}

	close(claim.messages)

	err := h.ConsumeClaim(mockSession, claim)
	require.NoError(t, err)
}

func TestConsumerGroupHandler_ConsumeClaim_CancelledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	mockSession := new(mockConsumerGroupSession)
	claim := &mockConsumerGroupClaim{
		messages: make(chan *sarama.ConsumerMessage),
	}

	h := &consumerGroupHandler{
		ctx:     ctx,
		handler: func(_ context.Context, _, _ []byte) error { return nil },
		logger:  zap.NewNop().Sugar(),
	}

	err := h.ConsumeClaim(mockSession, claim)
	require.NoError(t, err)
}

func TestConsumerGroupHandler_ConsumeClaim_RecoverPanic(t *testing.T) {
	t.Parallel()

	mockSession := new(mockConsumerGroupSession)
	claim := &mockConsumerGroupClaim{
		messages: make(chan *sarama.ConsumerMessage, 1),
	}

	h := &consumerGroupHandler{
		ctx: context.Background(),
		handler: func(_ context.Context, _, _ []byte) error {
			panic("test panic")
		},
		logger: zap.NewNop().Sugar(),
	}

	mockSession.On("MarkMessage", mock.Anything, "").Return()

	claim.messages <- &sarama.ConsumerMessage{Key: nil, Value: nil}

	close(claim.messages)

	err := h.ConsumeClaim(mockSession, claim)
	require.NoError(t, err)
}

type mockConsumerGroup struct {
	mock.Mock
}

func (m *mockConsumerGroup) Consume(ctx context.Context, topics []string, handler sarama.ConsumerGroupHandler) error {
	return m.Called(ctx, topics, handler).Error(0)
}

func (m *mockConsumerGroup) Errors() <-chan error { return m.Called().Get(0).(<-chan error) }

func (m *mockConsumerGroup) Close() error                              { return m.Called().Error(0) }
func (m *mockConsumerGroup) Pause(topicPartitions map[string][]int32)  { m.Called(topicPartitions) }
func (m *mockConsumerGroup) PauseAll()                                 { m.Called() }
func (m *mockConsumerGroup) Resume(topicPartitions map[string][]int32) { m.Called(topicPartitions) }
func (m *mockConsumerGroup) ResumeAll()                                { m.Called() }

func TestConsumer_StartStop(t *testing.T) {
	t.Parallel()

	mockCG := new(mockConsumerGroup)
	mockCG.On("Consume", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	var closeCalled bool

	mockCG.On("Close").Return(nil).Run(func(mock.Arguments) {
		closeCalled = true
	})

	c := &Consumer{
		client:  mockCG,
		topic:   "test-topic",
		groupID: "test-group",
		handler: func(_ context.Context, _, _ []byte) error { return nil },
		logger:  zap.NewNop().Sugar(),
	}

	err := c.Start(context.Background())
	require.NoError(t, err)
	assert.True(t, c.started)

	err = c.Start(context.Background())
	require.NoError(t, err)

	err = c.Stop(context.Background())
	require.NoError(t, err)
	assert.False(t, c.started)

	mockCG.AssertExpectations(t)
	assert.True(t, closeCalled)
}

func TestConsumer_StopWithoutStart(t *testing.T) {
	t.Parallel()

	mockCG := new(mockConsumerGroup)

	c := &Consumer{
		client:  mockCG,
		topic:   "test-topic",
		groupID: "test-group",
		logger:  zap.NewNop().Sugar(),
	}

	err := c.Stop(context.Background())
	require.NoError(t, err)
}

type mockClusterAdmin struct {
	mock.Mock
}

func (m *mockClusterAdmin) CreateTopic(topic string, detail *sarama.TopicDetail, validateOnly bool) error {
	return m.Called(topic, detail, validateOnly).Error(0)
}

func (m *mockClusterAdmin) ListTopics() (map[string]sarama.TopicDetail, error) {
	args := m.Called()
	return args.Get(0).(map[string]sarama.TopicDetail), args.Error(1)
}

func (m *mockClusterAdmin) DescribeTopics(topics []string) ([]*sarama.TopicMetadata, error) {
	args := m.Called(topics)
	return args.Get(0).([]*sarama.TopicMetadata), args.Error(1)
}

func (m *mockClusterAdmin) DeleteTopic(topic string) error {
	return m.Called(topic).Error(0)
}

func (m *mockClusterAdmin) CreatePartitions(topic string, count int32, assignment [][]int32, validateOnly bool) error {
	return m.Called(topic, count, assignment, validateOnly).Error(0)
}

func (m *mockClusterAdmin) AlterPartitionReassignments(topic string, assignment [][]int32) error {
	return m.Called(topic, assignment).Error(0)
}

func (m *mockClusterAdmin) ListPartitionReassignments(
	topics string,
	partitions []int32,
) (map[string]map[int32]*sarama.PartitionReplicaReassignmentsStatus, error) {
	args := m.Called(topics, partitions)
	return args.Get(0).(map[string]map[int32]*sarama.PartitionReplicaReassignmentsStatus), args.Error(1)
}

func (m *mockClusterAdmin) DeleteRecords(topic string, partitionOffsets map[int32]int64) error {
	return m.Called(topic, partitionOffsets).Error(0)
}

func (m *mockClusterAdmin) DescribeConfig(resource sarama.ConfigResource) ([]sarama.ConfigEntry, error) {
	args := m.Called(resource)
	return args.Get(0).([]sarama.ConfigEntry), args.Error(1)
}

func (m *mockClusterAdmin) DescribeConfigs(
	resources []*sarama.ConfigResource,
	options sarama.DescribeConfigsOptions,
) ([]*sarama.ConfigResourceResult, error) {
	args := m.Called(resources, options)
	return args.Get(0).([]*sarama.ConfigResourceResult), args.Error(1)
}

func (m *mockClusterAdmin) AlterConfig(
	resourceType sarama.ConfigResourceType,
	name string,
	entries map[string]*string,
	validateOnly bool,
) error {
	return m.Called(resourceType, name, entries, validateOnly).Error(0)
}

func (m *mockClusterAdmin) IncrementalAlterConfig(
	resourceType sarama.ConfigResourceType,
	name string,
	entries map[string]sarama.IncrementalAlterConfigsEntry,
	validateOnly bool,
) error {
	return m.Called(resourceType, name, entries, validateOnly).Error(0)
}

func (m *mockClusterAdmin) CreateACL(resource sarama.Resource, acl sarama.Acl) error {
	return m.Called(resource, acl).Error(0)
}

func (m *mockClusterAdmin) CreateACLs(resourceAcls []*sarama.ResourceAcls) error {
	return m.Called(resourceAcls).Error(0)
}

func (m *mockClusterAdmin) ListAcls(filter sarama.AclFilter) ([]sarama.ResourceAcls, error) {
	args := m.Called(filter)
	return args.Get(0).([]sarama.ResourceAcls), args.Error(1)
}

func (m *mockClusterAdmin) DeleteACL(filter sarama.AclFilter, validateOnly bool) ([]sarama.MatchingAcl, error) {
	args := m.Called(filter, validateOnly)
	return args.Get(0).([]sarama.MatchingAcl), args.Error(1)
}

func (m *mockClusterAdmin) ElectLeaders(
	electionType sarama.ElectionType,
	partitions map[string][]int32,
) (map[string]map[int32]*sarama.PartitionResult, error) {
	args := m.Called(electionType, partitions)
	return args.Get(0).(map[string]map[int32]*sarama.PartitionResult), args.Error(1)
}

func (m *mockClusterAdmin) ListConsumerGroups() (map[string]string, error) {
	args := m.Called()
	return args.Get(0).(map[string]string), args.Error(1)
}

func (m *mockClusterAdmin) DescribeConsumerGroups(groups []string) ([]*sarama.GroupDescription, error) {
	args := m.Called(groups)
	return args.Get(0).([]*sarama.GroupDescription), args.Error(1)
}

func (m *mockClusterAdmin) ListConsumerGroupOffsets(group string, topicPartitions map[string][]int32) (*sarama.OffsetFetchResponse, error) {
	args := m.Called(group, topicPartitions)
	return args.Get(0).(*sarama.OffsetFetchResponse), args.Error(1)
}

func (m *mockClusterAdmin) ListConsumerGroupOffsetsBatch(
	groupTopics map[string]map[string][]int32,
) (map[string]*sarama.OffsetFetchResponseGroup, error) {
	args := m.Called(groupTopics)
	return args.Get(0).(map[string]*sarama.OffsetFetchResponseGroup), args.Error(1)
}

func (m *mockClusterAdmin) ListOffsets(
	partitions map[string]map[int32]int64,
	options *sarama.ListOffsetsOptions,
) (map[string]map[int32]*sarama.OffsetResult, error) {
	args := m.Called(partitions, options)
	return args.Get(0).(map[string]map[int32]*sarama.OffsetResult), args.Error(1)
}

func (m *mockClusterAdmin) AlterConsumerGroupOffsets(
	group string,
	offsets map[string]map[int32]sarama.OffsetAndMetadata,
	options *sarama.AlterConsumerGroupOffsetsOptions,
) (*sarama.OffsetCommitResponse, error) {
	args := m.Called(group, offsets, options)
	return args.Get(0).(*sarama.OffsetCommitResponse), args.Error(1)
}

func (m *mockClusterAdmin) DeleteConsumerGroupOffset(group string, topic string, partition int32) error {
	return m.Called(group, topic, partition).Error(0)
}

func (m *mockClusterAdmin) DeleteConsumerGroup(group string) error {
	return m.Called(group).Error(0)
}

func (m *mockClusterAdmin) DescribeCluster() ([]*sarama.Broker, int32, error) {
	args := m.Called()
	return args.Get(0).([]*sarama.Broker), args.Get(1).(int32), args.Error(2)
}

func (m *mockClusterAdmin) DescribeLogDirs(brokers []int32) (map[int32][]sarama.DescribeLogDirsResponseDirMetadata, error) {
	args := m.Called(brokers)
	return args.Get(0).(map[int32][]sarama.DescribeLogDirsResponseDirMetadata), args.Error(1)
}

func (m *mockClusterAdmin) DescribeUserScramCredentials(users []string) ([]*sarama.DescribeUserScramCredentialsResult, error) {
	args := m.Called(users)
	return args.Get(0).([]*sarama.DescribeUserScramCredentialsResult), args.Error(1)
}

func (m *mockClusterAdmin) DeleteUserScramCredentials(
	deletes []sarama.AlterUserScramCredentialsDelete,
) ([]*sarama.AlterUserScramCredentialsResult, error) {
	args := m.Called(deletes)
	return args.Get(0).([]*sarama.AlterUserScramCredentialsResult), args.Error(1)
}

func (m *mockClusterAdmin) UpsertUserScramCredentials(
	upsert []sarama.AlterUserScramCredentialsUpsert,
) ([]*sarama.AlterUserScramCredentialsResult, error) {
	args := m.Called(upsert)
	return args.Get(0).([]*sarama.AlterUserScramCredentialsResult), args.Error(1)
}

func (m *mockClusterAdmin) UpdateFeatures(featureUpdates []sarama.FeatureUpdate) ([]sarama.UpdatableFeatureResult, error) {
	args := m.Called(featureUpdates)
	return args.Get(0).([]sarama.UpdatableFeatureResult), args.Error(1)
}

func (m *mockClusterAdmin) DescribeClientQuotas(
	components []sarama.QuotaFilterComponent,
	strict bool,
) ([]sarama.DescribeClientQuotasEntry, error) {
	args := m.Called(components, strict)
	return args.Get(0).([]sarama.DescribeClientQuotasEntry), args.Error(1)
}

func (m *mockClusterAdmin) AlterClientQuotas(entity []sarama.QuotaEntityComponent, op sarama.ClientQuotasOp, validateOnly bool) error {
	return m.Called(entity, op, validateOnly).Error(0)
}

func (m *mockClusterAdmin) Controller() (*sarama.Broker, error) {
	args := m.Called()
	return args.Get(0).(*sarama.Broker), args.Error(1)
}

func (m *mockClusterAdmin) Coordinator(group string) (*sarama.Broker, error) {
	args := m.Called(group)
	return args.Get(0).(*sarama.Broker), args.Error(1)
}

func (m *mockClusterAdmin) RemoveMemberFromConsumerGroup(groupID string, groupInstanceIDs []string) (*sarama.LeaveGroupResponse, error) {
	args := m.Called(groupID, groupInstanceIDs)
	return args.Get(0).(*sarama.LeaveGroupResponse), args.Error(1)
}

func (m *mockClusterAdmin) Close() error {
	return m.Called().Error(0)
}

type mockClient struct {
	mock.Mock
}

func (m *mockClient) Config() *sarama.Config { return m.Called().Get(0).(*sarama.Config) }
func (m *mockClient) Controller() (*sarama.Broker, error) {
	a := m.Called()
	return a.Get(0).(*sarama.Broker), a.Error(1)
}

func (m *mockClient) RefreshController() (*sarama.Broker, error) {
	a := m.Called()
	return a.Get(0).(*sarama.Broker), a.Error(1)
}
func (m *mockClient) Brokers() []*sarama.Broker { return m.Called().Get(0).([]*sarama.Broker) }
func (m *mockClient) Broker(brokerID int32) (*sarama.Broker, error) {
	a := m.Called(brokerID)
	return a.Get(0).(*sarama.Broker), a.Error(1)
}

func (m *mockClient) Topics() ([]string, error) {
	a := m.Called()
	return a.Get(0).([]string), a.Error(1)
}

func (m *mockClient) Partitions(topic string) ([]int32, error) {
	a := m.Called(topic)
	return a.Get(0).([]int32), a.Error(1)
}

func (m *mockClient) WritablePartitions(topic string) ([]int32, error) {
	a := m.Called(topic)
	return a.Get(0).([]int32), a.Error(1)
}

func (m *mockClient) Leader(topic string, partitionID int32) (*sarama.Broker, error) {
	a := m.Called(topic, partitionID)
	return a.Get(0).(*sarama.Broker), a.Error(1)
}

func (m *mockClient) LeaderAndEpoch(topic string, partitionID int32) (*sarama.Broker, int32, error) {
	a := m.Called(topic, partitionID)
	return a.Get(0).(*sarama.Broker), a.Get(1).(int32), a.Error(2)
}

func (m *mockClient) Replicas(topic string, partitionID int32) ([]int32, error) {
	a := m.Called(topic, partitionID)
	return a.Get(0).([]int32), a.Error(1)
}

func (m *mockClient) InSyncReplicas(topic string, partitionID int32) ([]int32, error) {
	a := m.Called(topic, partitionID)
	return a.Get(0).([]int32), a.Error(1)
}

func (m *mockClient) OfflineReplicas(topic string, partitionID int32) ([]int32, error) {
	a := m.Called(topic, partitionID)
	return a.Get(0).([]int32), a.Error(1)
}
func (m *mockClient) RefreshBrokers(addrs []string) error    { return m.Called(addrs).Error(0) }
func (m *mockClient) RefreshMetadata(topics ...string) error { return m.Called(topics).Error(0) }
func (m *mockClient) GetOffset(topic string, partitionID int32, time int64) (int64, error) {
	a := m.Called(topic, partitionID, time)
	return a.Get(0).(int64), a.Error(1)
}

func (m *mockClient) Coordinator(consumerGroup string) (*sarama.Broker, error) {
	a := m.Called(consumerGroup)
	return a.Get(0).(*sarama.Broker), a.Error(1)
}

func (m *mockClient) RefreshCoordinator(consumerGroup string) error {
	return m.Called(consumerGroup).Error(0)
}

func (m *mockClient) TransactionCoordinator(transactionID string) (*sarama.Broker, error) {
	a := m.Called(transactionID)
	return a.Get(0).(*sarama.Broker), a.Error(1)
}

func (m *mockClient) RefreshTransactionCoordinator(transactionID string) error {
	return m.Called(transactionID).Error(0)
}

func (m *mockClient) InitProducerID() (*sarama.InitProducerIDResponse, error) {
	a := m.Called()
	return a.Get(0).(*sarama.InitProducerIDResponse), a.Error(1)
}

func (m *mockClient) LeastLoadedBroker() *sarama.Broker { return m.Called().Get(0).(*sarama.Broker) }

func (m *mockClient) PartitionNotReadable(topic string, partition int32) bool {
	return m.Called(topic, partition).Bool(0)
}
func (m *mockClient) Close() error { return m.Called().Error(0) }
func (m *mockClient) Closed() bool { return m.Called().Bool(0) }

func TestAdminClient_ListConsumerGroups(t *testing.T) {
	t.Parallel()

	mockAdmin := new(mockClusterAdmin)
	mockAdmin.On("ListConsumerGroups").Return(map[string]string{
		"group-c": "Stable",
		"group-a": "Empty",
		"group-b": "Stable",
	}, nil)

	a := &adminClient{
		admin:  mockAdmin,
		client: new(mockClient),
		logger: zap.NewNop().Sugar(),
	}

	groups, err := a.ListConsumerGroups()
	require.NoError(t, err)

	require.Len(t, groups, 3)
	assert.Equal(t, "group-a", groups[0].GroupID)
	assert.Equal(t, "group-b", groups[1].GroupID)
	assert.Equal(t, "group-c", groups[2].GroupID)
}

// validMemberAssignmentV0 is the V0-encoded ConsumerGroupMemberAssignment
// with Topics: {"topic-a": {0}} and no UserData.
// Format: Version(int16) | topicCount(int32) | topicLen(int16) | topic([]byte) |
//
//	partCount(int32) | part0(int64) | userDataLen(int32)
var validMemberAssignmentV0 = []byte{
	0, 0, // Version = 0 (int16)
	0, 0, 0, 1, // 1 topic (int32)
	0, 7, // topic string length (int16)
	't', 'o', 'p', 'i', 'c', '-', 'a', // "topic-a"
	0, 0, 0, 1, // 1 partition (int32)
	0, 0, 0, 0, // partition 0 (int32)
	255, 255, 255, 255, // UserData = nil (int32 -1)
}

func TestAdminClient_DescribeConsumerGroup(t *testing.T) {
	t.Parallel()

	mockAdmin := new(mockClusterAdmin)
	mockCl := new(mockClient)

	desc := &sarama.GroupDescription{
		GroupId: "test-group",
		State:   "Stable",
		Members: map[string]*sarama.GroupMemberDescription{
			"m1": {
				MemberId:         "m1",
				ClientId:         "c1",
				MemberAssignment: validMemberAssignmentV0,
			},
		},
	}

	mockAdmin.On("DescribeConsumerGroups", []string{"test-group"}).Return([]*sarama.GroupDescription{desc}, nil)
	mockAdmin.On("ListConsumerGroupOffsets", "test-group", mock.Anything).Return(&sarama.OffsetFetchResponse{
		Blocks: map[string]map[int32]*sarama.OffsetFetchResponseBlock{
			"topic-a": {
				0: {Offset: 10, Err: sarama.ErrNoError},
			},
		},
	}, nil)
	mockCl.On("GetOffset", "topic-a", int32(0), sarama.OffsetNewest).Return(int64(100), nil)

	a := &adminClient{
		admin:  mockAdmin,
		client: mockCl,
		logger: zap.NewNop().Sugar(),
	}

	detail, err := a.DescribeConsumerGroup("test-group")
	require.NoError(t, err)
	assert.Equal(t, "test-group", detail.GroupID)
	assert.Equal(t, "Stable", detail.State)

	require.Len(t, detail.Topics, 1)
	assert.Equal(t, "topic-a", detail.Topics[0].Topic)

	require.Len(t, detail.Topics[0].Partitions, 1)
	assert.Equal(t, int32(0), detail.Topics[0].Partitions[0].Partition)
	assert.Equal(t, int64(10), detail.Topics[0].Partitions[0].CurrentOffset)
	assert.Equal(t, int64(100), detail.Topics[0].Partitions[0].EndOffset)
	assert.Equal(t, int64(90), detail.Topics[0].Partitions[0].Lag)
}

func TestAdminClient_DescribeConsumerGroup_NotFound(t *testing.T) {
	t.Parallel()

	mockAdmin := new(mockClusterAdmin)
	mockAdmin.On("DescribeConsumerGroups", []string{"nonexistent"}).Return([]*sarama.GroupDescription{}, nil)

	a := &adminClient{
		admin:  mockAdmin,
		client: new(mockClient),
		logger: zap.NewNop().Sugar(),
	}

	_, err := a.DescribeConsumerGroup("nonexistent")
	assert.ErrorContains(t, err, "not found")
}

func TestAdminClient_DescribeConsumerGroup_NoMembers(t *testing.T) {
	t.Parallel()

	mockAdmin := new(mockClusterAdmin)

	desc := &sarama.GroupDescription{
		GroupId: "test-group",
		State:   "Stable",
	}
	mockAdmin.On("DescribeConsumerGroups", []string{"test-group"}).Return([]*sarama.GroupDescription{desc}, nil)

	a := &adminClient{
		admin:  mockAdmin,
		client: new(mockClient),
		logger: zap.NewNop().Sugar(),
	}

	detail, err := a.DescribeConsumerGroup("test-group")
	require.NoError(t, err)
	assert.Equal(t, "test-group", detail.GroupID)
	assert.Empty(t, detail.Topics)
}

func TestAdminClient_DescribeConsumerGroup_NilMemberAssignment(t *testing.T) {
	t.Parallel()

	mockAdmin := new(mockClusterAdmin)

	desc := &sarama.GroupDescription{
		GroupId: "test-group",
		State:   "Stable",
		Members: map[string]*sarama.GroupMemberDescription{
			"m1": {
				MemberId:         "m1",
				ClientId:         "c1",
				MemberAssignment: nil,
			},
		},
	}
	mockAdmin.On("DescribeConsumerGroups", []string{"test-group"}).Return([]*sarama.GroupDescription{desc}, nil)

	a := &adminClient{
		admin:  mockAdmin,
		client: new(mockClient),
		logger: zap.NewNop().Sugar(),
	}

	detail, err := a.DescribeConsumerGroup("test-group")
	require.NoError(t, err)
	assert.Equal(t, "test-group", detail.GroupID)
	assert.Empty(t, detail.Topics)
}

func TestAdminClient_DescribeConsumerGroup_BadMemberAssignment(t *testing.T) {
	t.Parallel()

	mockAdmin := new(mockClusterAdmin)

	desc := &sarama.GroupDescription{
		GroupId: "test-group",
		State:   "Stable",
		Members: map[string]*sarama.GroupMemberDescription{
			"m1": {
				MemberId:         "m1",
				ClientId:         "c1",
				MemberAssignment: []byte{0, 0, 0}, // truncated, will fail to decode
			},
		},
	}
	mockAdmin.On("DescribeConsumerGroups", []string{"test-group"}).Return([]*sarama.GroupDescription{desc}, nil)

	a := &adminClient{
		admin:  mockAdmin,
		client: new(mockClient),
		logger: zap.NewNop().Sugar(),
	}

	detail, err := a.DescribeConsumerGroup("test-group")
	require.NoError(t, err)
	assert.Equal(t, "test-group", detail.GroupID)
	assert.Empty(t, detail.Topics)
}

func TestAdminClient_DescribeConsumerGroup_ListConsumerGroupOffsetsError(t *testing.T) {
	t.Parallel()

	mockAdmin := new(mockClusterAdmin)

	desc := &sarama.GroupDescription{
		GroupId: "test-group",
		State:   "Stable",
		Members: map[string]*sarama.GroupMemberDescription{
			"m1": {
				MemberId:         "m1",
				ClientId:         "c1",
				MemberAssignment: validMemberAssignmentV0,
			},
		},
	}

	mockAdmin.On("DescribeConsumerGroups", []string{"test-group"}).Return([]*sarama.GroupDescription{desc}, nil)
	mockAdmin.On("ListConsumerGroupOffsets", "test-group", mock.Anything).Return(&sarama.OffsetFetchResponse{}, assert.AnError)

	a := &adminClient{
		admin:  mockAdmin,
		client: new(mockClient),
		logger: zap.NewNop().Sugar(),
	}

	detail, err := a.DescribeConsumerGroup("test-group")
	require.NoError(t, err)
	require.Len(t, detail.Topics, 1, "topic should be present even when ListConsumerGroupOffsets fails")
	assert.Empty(t, detail.Topics[0].Partitions)
}

func TestAdminClient_DescribeConsumerGroup_GetOffsetError(t *testing.T) {
	t.Parallel()

	mockAdmin := new(mockClusterAdmin)
	mockCl := new(mockClient)

	desc := &sarama.GroupDescription{
		GroupId: "test-group",
		State:   "Stable",
		Members: map[string]*sarama.GroupMemberDescription{
			"m1": {
				MemberId:         "m1",
				ClientId:         "c1",
				MemberAssignment: validMemberAssignmentV0,
			},
		},
	}

	mockAdmin.On("DescribeConsumerGroups", []string{"test-group"}).Return([]*sarama.GroupDescription{desc}, nil)
	mockAdmin.On("ListConsumerGroupOffsets", "test-group", mock.Anything).Return(&sarama.OffsetFetchResponse{
		Blocks: map[string]map[int32]*sarama.OffsetFetchResponseBlock{
			"topic-a": {
				0: {Offset: 10, Err: sarama.ErrNoError},
			},
		},
	}, nil)
	mockCl.On("GetOffset", "topic-a", int32(0), sarama.OffsetNewest).Return(int64(0), assert.AnError)

	a := &adminClient{
		admin:  mockAdmin,
		client: mockCl,
		logger: zap.NewNop().Sugar(),
	}

	detail, err := a.DescribeConsumerGroup("test-group")
	require.NoError(t, err)
	require.Len(t, detail.Topics, 1, "topic should be present even when GetOffset fails")
	assert.Empty(t, detail.Topics[0].Partitions)
}

func TestAdminClient_Close(t *testing.T) {
	t.Parallel()

	mockAdmin := new(mockClusterAdmin)
	mockCl := new(mockClient)

	mockAdmin.On("Close").Return(nil)
	mockCl.On("Close").Return(nil)

	a := &adminClient{
		admin:  mockAdmin,
		client: mockCl,
		logger: zap.NewNop().Sugar(),
	}

	err := a.Close()
	require.NoError(t, err)
}

func TestProducer_HandleErrors(t *testing.T) {
	t.Parallel()

	errChan := make(chan *sarama.ProducerError, 1)
	mockProd := new(mockAsyncProducer)
	mockProd.On("Errors").Return(errChan)

	p := &Producer{
		producer: mockProd,
		logger:   zap.NewNop().Sugar(),
	}

	errChan <- &sarama.ProducerError{
		Msg: &sarama.ProducerMessage{Topic: "test", Key: sarama.StringEncoder("key")},
		Err: assert.AnError,
	}

	close(errChan)

	p.handleErrors()
	mockProd.AssertExpectations(t)
}
