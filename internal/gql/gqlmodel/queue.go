package gqlmodel

import (
	"context"
	"fmt"
	"time"

	"github.com/hexsans/hexmagnet/internal/gql/gqlmodel/gen"
	"github.com/hexsans/hexmagnet/internal/metrics/queuemetrics"
	"github.com/hexsans/hexmagnet/internal/queue"
)

type QueueQuery struct {
	QueueManager queue.Manager
}

func (q QueueQuery) Metrics(ctx context.Context, input gen.QueueMetricsQueryInput) (gen.QueueMetricsQueryResult, error) {
	var bucketDuration time.Duration

	switch input.BucketDuration {
	case gen.MetricsBucketDurationMinute:
		bucketDuration = time.Minute
	case gen.MetricsBucketDurationHour:
		bucketDuration = time.Hour
	case gen.MetricsBucketDurationDay:
		bucketDuration = 24 * time.Hour
	}

	mq := queue.MetricsQuery{
		BucketDuration: bucketDuration,
	}

	if queues, ok := input.Queues.ValueOK(); ok {
		mq.Queues = queues
	}

	if statuses, ok := input.Statuses.ValueOK(); ok {
		mq.Statuses = make([]string, len(statuses))
		for i, s := range statuses {
			mq.Statuses[i] = string(s)
		}
	}

	if t, ok := input.StartTime.ValueOK(); ok && t != nil {
		mq.StartTime = *t
	}

	if t, ok := input.EndTime.ValueOK(); ok && t != nil {
		mq.EndTime = *t
	}

	result, err := q.QueueManager.Metrics(ctx, mq)
	if err != nil {
		return gen.QueueMetricsQueryResult{}, fmt.Errorf("queue metrics: %w", err)
	}

	buckets := make([]queuemetrics.Bucket, len(result.Buckets))
	for i, b := range result.Buckets {
		buckets[i] = queuemetrics.Bucket{
			Queue:           b.Queue,
			Status:          b.Status,
			CreatedAtBucket: b.CreatedAtBucket,
			RanAtBucket:     b.RanAtBucket,
			Count:           b.Count,
			Latency:         b.Latency,
		}
	}

	return gen.QueueMetricsQueryResult{Buckets: buckets}, nil
}

func (q QueueQuery) Jobs(ctx context.Context, input gen.QueueJobsQueryInput) (gen.QueueJobsQueryResult, error) {
	jq := queue.JobsQuery{}

	if queues, ok := input.Queues.ValueOK(); ok && queues != nil {
		jq.Queues = &queue.FacetFilter{
			Values: queues.Values,
			Logic:  string(queues.Logic),
		}
	}

	if orderBy, ok := input.OrderBy.ValueOK(); ok && orderBy != nil {
		var desc bool
		if orderBy.Descending.IsSet() && orderBy.Descending.Value() != nil {
			desc = *orderBy.Descending.Value()
		}

		jq.OrderBy = &queue.JobsOrderBy{
			Field:      string(orderBy.Field),
			Descending: desc,
		}
	}

	if limit, ok := input.Limit.ValueOK(); ok && limit != nil {
		jq.Limit = *limit
	}

	if offset, ok := input.Offset.ValueOK(); ok && offset != nil {
		jq.Offset = *offset
	}

	result, err := q.QueueManager.Jobs(ctx, jq)
	if err != nil {
		return gen.QueueJobsQueryResult{}, fmt.Errorf("queue jobs: %w", err)
	}

	items := make([]gen.QueueJob, len(result.Items))
	for i, job := range result.Items {
		items[i] = gen.QueueJob{
			ID:        job.ID,
			Queue:     job.Queue,
			Payload:   job.Payload,
			CreatedAt: job.CreatedAt,
		}
	}

	queueAggs := make([]gen.QueueJobQueueAgg, len(result.Aggregations.Queue))
	for i, a := range result.Aggregations.Queue {
		queueAggs[i] = gen.QueueJobQueueAgg{
			Value: a.Value,
			Label: a.Label,
			Count: a.Count,
		}
	}

	return gen.QueueJobsQueryResult{
		TotalCount:  result.TotalCount,
		HasNextPage: &result.HasNextPage,
		Items:       items,
		Aggregations: gen.QueueJobsAggregations{
			Queue: queueAggs,
		},
	}, nil
}
