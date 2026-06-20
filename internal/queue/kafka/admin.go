package kafka

import (
	"fmt"
	"sort"

	"github.com/IBM/sarama"
	"go.uber.org/zap"
)

type ConsumerGroupSummary struct {
	GroupID string
	State   string
}

type PartitionOffset struct {
	Partition     int32
	CurrentOffset int64
	EndOffset     int64
	Lag           int64
}

type TopicDetail struct {
	Topic      string
	Partitions []PartitionOffset
}

type GroupDetail struct {
	GroupID string
	State   string
	Topics  []TopicDetail
}

type TopicInfo struct {
	Name string
}

type AdminClient interface {
	ListConsumerGroups() ([]ConsumerGroupSummary, error)
	DescribeConsumerGroup(groupID string) (*GroupDetail, error)
	ListTopics() ([]TopicInfo, error)
	Close() error
}

type adminClient struct {
	admin  sarama.ClusterAdmin
	client sarama.Client
	logger *zap.SugaredLogger
}

func NewAdminClient(brokers []string, logger *zap.SugaredLogger) (AdminClient, error) {
	config := sarama.NewConfig()
	config.Version = sarama.V2_6_0_0
	config.Consumer.Offsets.Initial = sarama.OffsetNewest

	admin, err := sarama.NewClusterAdmin(brokers, config)
	if err != nil {
		return nil, fmt.Errorf("create cluster admin: %w", err)
	}

	client, err := sarama.NewClient(brokers, config)
	if err != nil {
		admin.Close()
		return nil, fmt.Errorf("create client: %w", err)
	}

	return &adminClient{
		admin:  admin,
		client: client,
		logger: logger,
	}, nil
}

func (a *adminClient) ListConsumerGroups() ([]ConsumerGroupSummary, error) {
	groups, err := a.admin.ListConsumerGroups()
	if err != nil {
		return nil, fmt.Errorf("list consumer groups: %w", err)
	}

	result := make([]ConsumerGroupSummary, 0, len(groups))
	for groupID, state := range groups {
		result = append(result, ConsumerGroupSummary{
			GroupID: groupID,
			State:   state,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].GroupID < result[j].GroupID
	})

	return result, nil
}

func (a *adminClient) DescribeConsumerGroup(groupID string) (*GroupDetail, error) {
	descriptions, err := a.admin.DescribeConsumerGroups([]string{groupID})
	if err != nil {
		return nil, fmt.Errorf("describe consumer group %s: %w", groupID, err)
	}

	if len(descriptions) == 0 {
		return nil, fmt.Errorf("consumer group %s not found", groupID)
	}

	desc := descriptions[0]

	detail := &GroupDetail{
		GroupID: groupID,
		State:   desc.State,
	}

	topicPartitions := map[string][]int32{}

	for _, member := range desc.Members {
		assignment, err := member.GetMemberAssignment()
		if err != nil {
			a.logger.Warnw("failed to get member assignment", "group", groupID, "member", member.MemberId, "error", err)
			continue
		}

		if assignment == nil {
			a.logger.Debugw("nil member assignment", "group", groupID, "member", member.MemberId)
			continue
		}

		for topic, partitions := range assignment.Topics {
			topicPartitions[topic] = append(topicPartitions[topic], partitions...)
		}
	}

	if len(topicPartitions) == 0 {
		return detail, nil
	}

	for topic, partitions := range topicPartitions {
		td := TopicDetail{Topic: topic}

		for _, partition := range partitions {
			currentOffset, err := a.admin.ListConsumerGroupOffsets(groupID, map[string][]int32{topic: {partition}})
			if err != nil {
				a.logger.Warnw("failed to get consumer offset",
					"group", groupID, "topic", topic, "partition", partition, "error", err)

				continue
			}

			var current int64

			if topicBlocks, ok := currentOffset.Blocks[topic]; ok {
				if block, ok := topicBlocks[partition]; ok && block != nil {
					current = block.Offset
				}
			}

			endOffset, err := a.client.GetOffset(topic, partition, sarama.OffsetNewest)
			if err != nil {
				a.logger.Warnw("failed to get log end offset",
					"topic", topic, "partition", partition, "error", err)

				continue
			}

			lag := max(endOffset-current, 0)

			td.Partitions = append(td.Partitions, PartitionOffset{
				Partition:     partition,
				CurrentOffset: current,
				EndOffset:     endOffset,
				Lag:           lag,
			})
		}

		sort.Slice(td.Partitions, func(i, j int) bool {
			return td.Partitions[i].Partition < td.Partitions[j].Partition
		})

		detail.Topics = append(detail.Topics, td)
	}

	sort.Slice(detail.Topics, func(i, j int) bool {
		return detail.Topics[i].Topic < detail.Topics[j].Topic
	})

	return detail, nil
}

func (a *adminClient) ListTopics() ([]TopicInfo, error) {
	topics, err := a.admin.ListTopics()
	if err != nil {
		return nil, fmt.Errorf("list topics: %w", err)
	}

	internalPrefixes := []string{"__"}

	result := make([]TopicInfo, 0, len(topics))
	for topic := range topics {
		isInternal := false

		for _, prefix := range internalPrefixes {
			if len(topic) >= len(prefix) && topic[:len(prefix)] == prefix {
				isInternal = true
				break
			}
		}

		if isInternal {
			continue
		}

		result = append(result, TopicInfo{Name: topic})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result, nil
}

func (a *adminClient) Close() error {
	if err := a.client.Close(); err != nil {
		a.logger.Errorw("error closing kafka client", "error", err)
	}

	return a.admin.Close()
}
