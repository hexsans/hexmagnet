package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"time"

	"github.com/hexsans/hexmagnet/internal/queue/kafka"
	"go.uber.org/zap"
)

type MetricsQuery struct {
	BucketDuration time.Duration
	Queues         []string
	Statuses       []string
	StartTime      time.Time
	EndTime        time.Time
}

type MetricsBucket struct {
	Queue           string
	Status          string
	CreatedAtBucket time.Time
	RanAtBucket     *time.Time
	Count           uint
	Latency         *time.Duration
}

type MetricsResult struct {
	Buckets []MetricsBucket
}

type FacetFilter struct {
	Values []string
	Logic  string
}

type JobsOrderBy struct {
	Field      string
	Descending bool
}

type JobsQuery struct {
	Queues  *FacetFilter
	OrderBy *JobsOrderBy
	Limit   int
	Offset  int
}

type Job struct {
	ID        string
	Queue     string
	Payload   string
	RunAfter  time.Time
	CreatedAt time.Time
}

type AggQueue struct {
	Value string
	Label string
	Count int
}

type Aggregations struct {
	Queue []AggQueue
}

type JobsResult struct {
	TotalCount   int
	HasNextPage  bool
	Items        []Job
	Aggregations Aggregations
}

type Manager interface {
	Metrics(ctx context.Context, query MetricsQuery) (MetricsResult, error)
	Jobs(ctx context.Context, query JobsQuery) (JobsResult, error)
	Close() error
}

type manager struct {
	adminClient kafka.AdminClient
	logger      *zap.SugaredLogger
}

func mapGroupState(state string) string {
	switch state {
	case "Stable":
		return "processed"
	case "Empty":
		return "pending"
	case "Dead":
		return "failed"
	case "PreparingRebalance", "CompletingRebalance":
		return "retry"
	default:
		return "pending"
	}
}

func (m *manager) Metrics(ctx context.Context, query MetricsQuery) (MetricsResult, error) {
	groups, err := m.adminClient.ListConsumerGroups()
	if err != nil {
		m.logger.Errorw("failed to list consumer groups for metrics", "error", err)
		return MetricsResult{Buckets: []MetricsBucket{}}, nil
	}

	var buckets []MetricsBucket

	now := time.Now().Truncate(query.BucketDuration)

	for _, group := range groups {
		if ctx.Err() != nil {
			return MetricsResult{}, ctx.Err()
		}

		status := mapGroupState(group.State)

		if len(query.Statuses) > 0 {
			matched := slices.Contains(query.Statuses, status)
			if !matched {
				continue
			}
		}

		detail, err := m.adminClient.DescribeConsumerGroup(group.GroupID)
		if err != nil {
			continue
		}

		for _, topic := range detail.Topics {
			if len(query.Queues) > 0 {
				matched := slices.Contains(query.Queues, topic.Topic)
				if !matched {
					continue
				}
			}

			queueName := group.GroupID + " / " + topic.Topic

			var totalLag uint
			for _, p := range topic.Partitions {
				totalLag += uint(p.Lag)
			}

			buckets = append(buckets, MetricsBucket{
				Queue:           queueName,
				Status:          status,
				CreatedAtBucket: now,
				Count:           totalLag,
			})
		}
	}

	sort.Slice(buckets, func(i, j int) bool {
		return buckets[i].Queue < buckets[j].Queue
	})

	return MetricsResult{Buckets: buckets}, nil
}

type partitionPayload struct {
	Lag int64 `json:"lag"`
}

func (m *manager) Jobs(ctx context.Context, query JobsQuery) (JobsResult, error) {
	groups, err := m.adminClient.ListConsumerGroups()
	if err != nil {
		m.logger.Errorw("failed to list consumer groups", "error", err)

		return JobsResult{
			HasNextPage: false,
			Items:       []Job{},
			Aggregations: Aggregations{
				Queue: []AggQueue{},
			},
		}, nil
	}

	m.logger.Debugw("listing queue jobs", "groupCount", len(groups))

	var items []Job

	seenTopics := map[string]bool{}
	queueAggs := map[string]int{}

	for _, group := range groups {
		if ctx.Err() != nil {
			return JobsResult{}, ctx.Err()
		}

		detail, err := m.adminClient.DescribeConsumerGroup(group.GroupID)
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
				m.logger.Warnw("failed to marshal job payload", "error", err)
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

	topics, err := m.adminClient.ListTopics()
	if err != nil {
		m.logger.Warnw("failed to list topics", "error", err)
	} else {
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
				m.logger.Warnw("failed to marshal job payload", "error", err)
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

func (m *manager) Close() error {
	return m.adminClient.Close()
}
