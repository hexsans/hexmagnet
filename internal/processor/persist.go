package processor

import (
	"context"
	"fmt"

	"github.com/hexsans/hexmagnet/internal/database/db"
	"github.com/hexsans/hexmagnet/internal/model"
)

type persistPayload struct {
	torrents []model.Torrent
}

func (c *processor) persist(ctx context.Context, payload persistPayload) error {
	contents := make([]model.Content, 0, len(payload.torrents))
	contentSeen := make(map[model.ContentRef]struct{}, len(payload.torrents))

	for _, t := range payload.torrents {
		if t.ContentID.Valid && t.Content.CreatedAt.IsZero() && t.Content.Source != "" && t.Content.ID != "" {
			ref := t.Content.Ref()
			if _, ok := contentSeen[ref]; !ok {
				contentSeen[ref] = struct{}{}

				contents = append(contents, t.Content)
			}
		}
	}

	tx, err := c.queries.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := c.queries.WithTx(tx)

	for i := range contents {
		if err := q.UpsertContent(ctx, db.ContentToUpsertParams(contents[i])); err != nil {
			return fmt.Errorf("upsert content: %w", err)
		}
	}

	for _, t := range payload.torrents {
		if err := q.UpdateTorrentContent(ctx, db.TorrentToUpdateContentParams(t)); err != nil {
			return fmt.Errorf("update torrent content: %w", err)
		}
	}

	return tx.Commit(ctx)
}
