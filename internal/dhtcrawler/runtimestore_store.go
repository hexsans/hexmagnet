package dhtcrawler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"runtime/debug"
	"sync"
	"time"

	"github.com/hexsans/hexmagnet/internal/database/db"
	"github.com/hexsans/hexmagnet/internal/utils"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

const (
	saveInterval           = 30 * time.Second
	bloomSaveInterval      = 3 * time.Minute
	keyCrawlerState        = "crawler.state"
	keySeenConnectedPeers  = "crawler.stat.seen_connected_peers"
	keySeenDiscoveredPeers = "crawler.stat.seen_discovered_peers"
)

type RuntimeState struct {
	PeersConnected  uint64          `json:"peers_connected"`
	PeersDiscovered uint64          `json:"peers_discovered"`
	UptimeSeconds   uint64          `json:"uptime_seconds"`
	Activity        []ActivityEntry `json:"activity"`
}

type Store struct {
	pool    utils.Lazy[*pgxpool.Pool]
	runtime *Runtime
	logger  *zap.SugaredLogger

	stop       chan struct{}
	stopCtx    context.Context
	stopCancel context.CancelFunc
	wg         sync.WaitGroup
}

func NewStore(pool utils.Lazy[*pgxpool.Pool], runtime *Runtime, logger *zap.SugaredLogger) *Store {
	return &Store{
		pool:    pool,
		runtime: runtime,
		logger:  logger.Named("runtimestore"),
		stop:    make(chan struct{}),
	}
}

func (s *Store) Load(ctx context.Context) error {
	pool, err := s.pool.Get()
	if err != nil {
		return err
	}

	q := db.NewQueries(pool)

	record, err := q.GetKeyValue(ctx, keyCrawlerState)
	switch {
	case err == nil:
		var state RuntimeState
		if err := json.Unmarshal(record.Value, &state); err != nil {
			s.logger.Warnw("failed to unmarshal crawler state, starting fresh", "error", err)
		} else {
			s.runtime.PeersConnected.Set(state.PeersConnected)
			s.runtime.PeersDiscovered.Set(state.PeersDiscovered)
			s.runtime.SetUptimeOffset(time.Duration(state.UptimeSeconds) * time.Second)
			s.runtime.SetActivity(state.Activity)

			s.logger.Infow("loaded crawler state",
				"peersConnected", state.PeersConnected,
				"peersDiscovered", state.PeersDiscovered,
				"uptimeSeconds", state.UptimeSeconds,
			)
		}
	case errors.Is(err, pgx.ErrNoRows):
		s.logger.Infow("no crawler state found, starting fresh")
	default:
		s.logger.Warnw("failed to read crawler state, starting fresh", "error", err)
	}

	s.loadBloom(ctx, pool, keySeenConnectedPeers, s.runtime.SeenConnectedPeers)
	s.loadBloom(ctx, pool, keySeenDiscoveredPeers, s.runtime.SeenDiscoveredPeers)

	return nil
}

func (s *Store) StartSaveLoop(ctx context.Context) {
	s.stopCtx, s.stopCancel = context.WithCancel(ctx)
	s.wg.Add(2)

	go func() {
		defer s.wg.Done()
		defer func() {
			if r := recover(); r != nil {
				s.logger.Errorw("runtime store stats save loop panicked",
					"panic", r,
					"stack", string(debug.Stack()),
				)
			}
		}()

		ticker := time.NewTicker(saveInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.save(ctx)
			case <-s.stop:
				s.save(ctx)
				return
			}
		}
	}()

	go func() {
		defer s.wg.Done()
		defer func() {
			if r := recover(); r != nil {
				s.logger.Errorw("runtime store bloom save loop panicked",
					"panic", r,
					"stack", string(debug.Stack()),
				)
			}
		}()

		ticker := time.NewTicker(bloomSaveInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.saveBlooms(s.stopCtx)
			case <-s.stop:
				s.saveBlooms(s.stopCtx)
				return
			}
		}
	}()
}

func (s *Store) Stop() {
	close(s.stop)

	if s.stopCancel != nil {
		s.stopCancel()
	}

	s.wg.Wait()
}

func (s *Store) save(ctx context.Context) {
	state := RuntimeState{
		PeersConnected:  s.runtime.PeersConnected.Get(),
		PeersDiscovered: s.runtime.PeersDiscovered.Get(),
		UptimeSeconds:   uint64(s.runtime.Uptime().Seconds()),
		Activity:        s.runtime.RecentActivity(),
	}

	data, err := json.Marshal(state)
	if err != nil {
		s.logger.Warnw("failed to marshal runtime state", "error", err)
		return
	}

	pool, err := s.pool.Get()
	if err != nil {
		s.logger.Warnw("failed to get pool for state save", "error", err)
		return
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	q := db.NewQueries(pool)
	if err := q.UpsertKeyValue(ctx, db.UpsertKeyValueParams{
		Key:   keyCrawlerState,
		Value: data,
	}); err != nil {
		s.logger.Warnw("failed to save crawler state", "error", err)
	}
}

func (s *Store) saveBlooms(ctx context.Context) {
	pool, err := s.pool.Get()
	if err != nil {
		s.logger.Errorw("failed to get pool for bloom save", "error", err)
		return
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := s.saveBloom(ctx, pool, keySeenConnectedPeers, s.runtime.SeenConnectedPeers); err != nil {
		s.logger.Errorw("failed to save connected peers bloom",
			"key", keySeenConnectedPeers, "error", err,
		)
	}

	if err := s.saveBloom(ctx, pool, keySeenDiscoveredPeers, s.runtime.SeenDiscoveredPeers); err != nil {
		s.logger.Errorw("failed to save discovered peers bloom",
			"key", keySeenDiscoveredPeers, "error", err,
		)
	}
}

func (s *Store) loadBloom(ctx context.Context, pool *pgxpool.Pool, key string, bloom *SeenPeersBloom) {
	q := db.NewQueries(pool)

	record, err := q.GetKeyValue(ctx, key)
	if errors.Is(err, pgx.ErrNoRows) {
		return
	}

	if err != nil {
		s.logger.Warnw("failed to get bloom filter", "key", key, "error", err)
		return
	}

	if len(record.Value) == 0 {
		return
	}

	if err := bloom.GobDecode(record.Value); err != nil {
		s.logger.Warnw("failed to restore bloom filter", "key", key, "error", err)
	}
}

func (*Store) saveBloom(ctx context.Context, pool *pgxpool.Pool, key string, bloom *SeenPeersBloom) error {
	var buf bytes.Buffer
	if _, err := bloom.WriteTo(&buf); err != nil {
		return err
	}

	q := db.NewQueries(pool)

	return q.UpsertKeyValue(ctx, db.UpsertKeyValueParams{
		Key:   key,
		Value: buf.Bytes(),
	})
}
