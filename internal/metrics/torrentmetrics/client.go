package torrentmetrics

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hexsans/hexmagnet/internal/database/db"
	"github.com/hexsans/hexmagnet/internal/metrics"
)

type Bucket struct {
	Bucket       time.Time
	Count        uint
	UpdatedCount uint
}

type Request struct {
	BucketDuration metrics.BucketDuration
	StartTime      time.Time
	EndTime        time.Time
}

type Client interface {
	Request(context.Context, Request) ([]Bucket, error)
}

type client struct {
	q *db.Queries
}

func (c client) Request(ctx context.Context, req Request) ([]Bucket, error) {
	var (
		conditions []string
		params     []any
	)

	idx := 1

	selects := fmt.Sprintf(
		`date_trunc($%d, updated_at) as bucket, COUNT(*) as count, `+
			`COUNT(*) FILTER (WHERE updated_at > created_at + interval '1 hour') as updated_count`,
		idx,
	)

	params = append(params, req.BucketDuration)
	idx++

	if !req.StartTime.IsZero() {
		conditions = append(conditions, fmt.Sprintf("updated_at >= $%d", idx))
		params = append(params, req.StartTime)
		idx++
	}

	if !req.EndTime.IsZero() {
		conditions = append(conditions, fmt.Sprintf("updated_at <= $%d", idx))
		params = append(params, req.EndTime)
	}

	conditionClause := ""
	if len(conditions) > 0 {
		conditionClause = "WHERE (" + strings.Join(conditions, " AND ") + ")"
	}

	query := fmt.Sprintf(`SELECT %s
FROM torrents
%s
GROUP BY bucket
ORDER BY bucket`, selects, conditionClause)

	var result []Bucket

	rows, err := c.q.Read(ctx).Query(ctx, query, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var b Bucket
		if err := rows.Scan(&b.Bucket, &b.Count, &b.UpdatedCount); err != nil {
			return nil, err
		}

		result = append(result, b)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}
