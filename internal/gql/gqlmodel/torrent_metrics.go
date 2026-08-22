package gqlmodel

import (
	"context"

	"github.com/hexsans/hexmagnet/internal/gql/gqlmodel/gen"
	"github.com/hexsans/hexmagnet/internal/metrics/torrentmetrics"
)

func (t TorrentQuery) Metrics(
	ctx context.Context,
	input gen.TorrentMetricsQueryInput,
) (*gen.TorrentMetricsQueryResult, error) {
	req := torrentmetrics.Request{
		StartTime: nilToZero(input.StartTime.Value()),
		EndTime:   nilToZero(input.EndTime.Value()),
		Timezone:  "UTC",
	}

	if tz, ok := input.Timezone.ValueOK(); ok && tz != nil && *tz != "" {
		req.Timezone = *tz
	}

	switch input.BucketDuration {
	case gen.MetricsBucketDurationMinute:
		req.BucketDuration = "minute"
	case gen.MetricsBucketDurationHour:
		req.BucketDuration = "hour"
	case gen.MetricsBucketDurationDay:
		req.BucketDuration = "day"
	}

	buckets, err := t.TorrentMetricsClient.Request(ctx, req)
	if err != nil {
		return nil, err
	}

	return &gen.TorrentMetricsQueryResult{
		Buckets: buckets,
	}, nil
}
