package gqlmodel

import (
	"context"
	"time"

	"github.com/hexsans/hexmagnet/internal/database/db"
	"github.com/hexsans/hexmagnet/internal/dhtcrawler"
	"github.com/hexsans/hexmagnet/internal/gql/gqlmodel/gen"
	"github.com/hexsans/hexmagnet/internal/model"
)

type DhtCrawlerStatus struct {
	Runtime     *dhtcrawler.Runtime
	DB          *db.Queries
	Paused      bool
	PauseReason model.NullString
}

func (s DhtCrawlerStatus) Active(_ context.Context) (bool, error) {
	return s.Runtime.Active.Get(), nil
}

func (s DhtCrawlerStatus) TorrentsCrawled(ctx context.Context) (uint64, error) {
	count, err := s.DB.CountTorrents(ctx)
	if err != nil {
		return 0, err
	}

	return uint64(count), nil
}

func (s DhtCrawlerStatus) PeersConnected(_ context.Context) (uint64, error) {
	return s.Runtime.PeersConnected.Get(), nil
}

func (s DhtCrawlerStatus) PeersDiscovered(_ context.Context) (uint64, error) {
	return s.Runtime.PeersDiscovered.Get(), nil
}

func (s DhtCrawlerStatus) Uptime(_ context.Context) (uint64, error) {
	return uint64(s.Runtime.Uptime().Seconds()), nil
}

func (s DhtCrawlerStatus) StartedAt(_ context.Context) (*time.Time, error) {
	t := s.Runtime.StartedAt.Get()
	if t.IsZero() {
		return nil, nil //nolint:nilnil // zero timestamp means the crawler never started
	}

	return &t, nil
}

func (s DhtCrawlerStatus) RecentActivity(_ context.Context) ([]gen.ActivityEntry, error) {
	entries := s.Runtime.RecentActivity()

	result := make([]gen.ActivityEntry, len(entries))
	for i, e := range entries {
		result[i] = gen.ActivityEntry{
			ID:      e.ID,
			Type:    e.Type,
			Message: e.Message,
			Time:    e.Time,
		}
	}

	return result, nil
}
