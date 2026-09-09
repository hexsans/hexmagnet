package gqlmodel

import (
	"context"

	"github.com/hexsans/hexmagnet/internal/gql/gqlmodel/gen"
	"github.com/hexsans/hexmagnet/internal/protocol"
	"github.com/hexsans/hexmagnet/internal/retryqueue"
)

type RetryQueueQuery struct {
	Queue *retryqueue.Queue
}

func (q RetryQueueQuery) Entries(ctx context.Context, input gen.RetryQueueEntriesInput) (gen.RetryQueueEntriesResult, error) {
	limit, offset := 50, 0

	if v, ok := input.Limit.ValueOK(); ok && v != nil {
		limit = *v
	}

	if v, ok := input.Offset.ValueOK(); ok && v != nil {
		offset = *v
	}

	entries, total, err := q.Queue.Entries(ctx, limit, offset)
	if err != nil {
		return gen.RetryQueueEntriesResult{}, err
	}

	items := make([]gen.RetryQueueEntry, len(entries))
	for i, e := range entries {
		infoHash, parseErr := protocol.ParseID(e.InfoHash)
		if parseErr != nil {
			return gen.RetryQueueEntriesResult{}, parseErr
		}

		items[i] = gen.RetryQueueEntry{
			InfoHash:      infoHash,
			Stage:         e.Stage,
			FailCount:     e.FailCount,
			LastError:     e.LastError,
			LastFailureAt: e.LastFailureAt,
			NextRetryAt:   e.NextRetryAt,
		}
	}

	return gen.RetryQueueEntriesResult{
		Total: uint64(total), //nolint:gosec // non-negative count
		Items: items,
	}, nil
}

type RetryQueueMutation struct {
	Queue *retryqueue.Queue
}

func (m RetryQueueMutation) Retry(ctx context.Context, infoHash protocol.ID) (bool, error) {
	if err := m.Queue.RetryNow(ctx, infoHash.String()); err != nil {
		return false, err
	}

	return true, nil
}

func (m RetryQueueMutation) Clear(ctx context.Context) error {
	return m.Queue.Clear(ctx)
}
