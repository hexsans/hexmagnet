package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"time"

	"github.com/hexsans/hexmagnet/internal/queue/memoryqueue"
)

type memoryManager struct {
	queue *memoryqueue.MemoryQueue
}

func (*memoryManager) Metrics(_ context.Context, _ MetricsQuery) (MetricsResult, error) {
	return MetricsResult{}, nil
}

func (m *memoryManager) Jobs(ctx context.Context, query JobsQuery) (JobsResult, error) {
	groups, err := m.queue.ListConsumerGroups()
	if err != nil {
		return JobsResult{}, err
	}

	var items []Job

	seenTopics := map[string]bool{}
	queueAggs := map[string]int{}

	for _, group := range groups {
		if ctx.Err() != nil {
			return JobsResult{}, ctx.Err()
		}

		detail, err := m.queue.DescribeConsumerGroup(group.GroupID)
		if err != nil {
			continue
		}

		for _, topic := range detail.Topics {
			if query.Queues != nil {
				matched := slices.Contains(query.Queues.Values, topic.Topic)
				if query.Queues.Logic == "or" && !matched {
					continue
				}
			}

			id := fmt.Sprintf("%s/%s", group.GroupID, topic.Topic)

			var totalLag int64
			for _, p := range topic.Partitions {
				totalLag += p.Lag
			}

			payload := partitionPayload{Lag: totalLag}

			payloadBytes, err := json.Marshal(payload)
			if err != nil {
				continue
			}

			items = append(items, Job{
				ID:        id,
				Queue:     topic.Topic,
				Payload:   string(payloadBytes),
				RunAfter:  time.Now(),
				CreatedAt: time.Now(),
			})

			seenTopics[topic.Topic] = true
			queueAggs[topic.Topic]++
		}
	}

	topics, err := m.queue.ListTopics()
	if err == nil {
		for _, topic := range topics {
			if seenTopics[topic.Name] {
				continue
			}

			if query.Queues != nil {
				matched := slices.Contains(query.Queues.Values, topic.Name)
				if query.Queues.Logic == "or" && !matched {
					continue
				}
			}

			payloadBytes, err := json.Marshal(partitionPayload{Lag: 0})
			if err != nil {
				continue
			}

			items = append(items, Job{
				ID:        fmt.Sprintf("pending/%s", topic.Name),
				Queue:     topic.Name,
				Payload:   string(payloadBytes),
				RunAfter:  time.Now(),
				CreatedAt: time.Now(),
			})

			queueAggs[topic.Name]++
		}
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})

	totalCount := len(items)
	offset := query.Offset

	limit := query.Limit
	if limit == 0 {
		limit = 20
	}

	if offset > totalCount {
		offset = totalCount
	}

	end := min(offset+limit, totalCount)
	page := items[offset:end]

	var queueAggList []AggQueue
	for q, c := range queueAggs {
		queueAggList = append(queueAggList, AggQueue{Value: q, Label: q, Count: c})
	}

	sort.Slice(queueAggList, func(i, j int) bool {
		return queueAggList[i].Value < queueAggList[j].Value
	})

	return JobsResult{
		TotalCount:  totalCount,
		HasNextPage: end < totalCount,
		Items:       page,
		Aggregations: Aggregations{
			Queue: queueAggList,
		},
	}, nil
}

func (*memoryManager) Close() error {
	return nil
}
