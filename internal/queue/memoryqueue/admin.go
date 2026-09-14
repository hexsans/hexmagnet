package memoryqueue

import (
	"sort"

	"github.com/hexsans/hexmagnet/internal/queue/kafka"
)

func (mq *MemoryQueue) ListConsumerGroups() ([]kafka.ConsumerGroupSummary, error) {
	mq.mu.RLock()
	defer mq.mu.RUnlock()

	seen := make(map[string]string)

	for _, ts := range mq.topics {
		for _, cs := range ts.Consumers {
			seen[cs.GroupID] = "Stable"
		}
	}

	result := make([]kafka.ConsumerGroupSummary, 0, len(seen))
	for groupID, state := range seen {
		result = append(result, kafka.ConsumerGroupSummary{
			GroupID: groupID,
			State:   state,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].GroupID < result[j].GroupID
	})

	return result, nil
}

func (mq *MemoryQueue) DescribeConsumerGroup(groupID string) (*kafka.GroupDetail, error) {
	mq.mu.RLock()
	defer mq.mu.RUnlock()

	detail := &kafka.GroupDetail{
		GroupID: groupID,
		State:   "Stable",
	}

	for topic, ts := range mq.topics {
		var found bool

		for _, cs := range ts.Consumers {
			if cs.GroupID == groupID {
				found = true
				break
			}
		}

		if !found {
			continue
		}

		offset := mq.offsets[groupID][topic]

		// Offsets track the next sequence to consume, matching Kafka's
		// committed-offset convention: EndOffset is the next sequence that
		// will be assigned, so lag is simply end minus current.
		endOffset := int64(ts.NextSeq)

		detail.Topics = append(detail.Topics, kafka.TopicDetail{
			Topic: topic,
			Partitions: []kafka.PartitionOffset{
				{
					Partition:     0,
					CurrentOffset: int64(offset),
					EndOffset:     endOffset,
					Lag:           max(endOffset-int64(offset), 0),
				},
			},
		})
	}

	sort.Slice(detail.Topics, func(i, j int) bool {
		return detail.Topics[i].Topic < detail.Topics[j].Topic
	})

	return detail, nil
}

func (mq *MemoryQueue) ListTopics() ([]kafka.TopicInfo, error) {
	mq.mu.RLock()
	defer mq.mu.RUnlock()

	result := make([]kafka.TopicInfo, 0, len(mq.topics))
	for topic := range mq.topics {
		result = append(result, kafka.TopicInfo{Name: topic})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result, nil
}
