package configmgr

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/hexsans/hexmagnet/internal/classifier"
	"github.com/hexsans/hexmagnet/internal/database/postgres"
	"github.com/hexsans/hexmagnet/internal/processor/enrich/indexer"
	"github.com/hexsans/hexmagnet/internal/protocol/dht"
	"github.com/hexsans/hexmagnet/internal/protocol/metainfo/metainforequester"
	"github.com/hexsans/hexmagnet/internal/queue"
	"github.com/hexsans/hexmagnet/internal/servercfg"
	"go.uber.org/zap"
)

type Snapshot struct {
	DHTRequester metainforequester.Config
	DHT          dht.Config
	Server       servercfg.Config
	Classifier   classifier.Config
	Postgres     postgres.Config
	Queue        queue.Config
	Search       indexer.SearchConfig
}

type ApplyMode int

const (
	ApplySync ApplyMode = iota
	ApplyAsync
)

type subscriber struct {
	name string
	fn   func(context.Context, *Snapshot) error
	mode ApplyMode
	ch   chan *Snapshot
}

type Manager struct {
	snapshot atomic.Pointer[Snapshot]
	subs     []*subscriber
	filePath string
	writeFn  func(path string, snap *Snapshot) error
	logger   *zap.SugaredLogger
	mu       sync.Mutex
	stopped  atomic.Bool
}

func NewManager(initial *Snapshot, filePath string, writeFn func(string, *Snapshot) error, logger *zap.SugaredLogger) *Manager {
	if initial == nil {
		initial = &Snapshot{}
	}

	m := &Manager{
		filePath: filePath,
		writeFn:  writeFn,
		logger:   logger,
	}
	m.snapshot.Store(initial)

	return m
}

func (m *Manager) Apply(ctx context.Context, snap *Snapshot) error {
	m.mu.Lock()
	subs := make([]*subscriber, len(m.subs))
	copy(subs, m.subs)
	m.mu.Unlock()

	// Sync subscribers run first: if any fail, the snapshot and file are
	// left untouched so the service keeps running on the old values.
	var applyErr error

	for _, s := range subs {
		if s.mode == ApplySync {
			if err := s.fn(ctx, snap); err != nil {
				applyErr = errors.Join(applyErr, fmt.Errorf("sync subscriber %q: %w", s.name, err))
			}
		}
	}

	if applyErr != nil {
		return applyErr
	}

	if m.writeFn != nil && m.filePath != "" {
		if err := m.writeFn(m.filePath, snap); err != nil {
			return fmt.Errorf("persist config: %w", err)
		}
	}

	m.snapshot.Store(snap)

	for _, s := range subs {
		if s.mode != ApplyAsync {
			continue
		}

		select {
		case s.ch <- snap:
		default:
			// Latest-wins: replace the pending snapshot so the newest value is applied.
			select {
			case <-s.ch:
			default:
			}

			select {
			case s.ch <- snap:
			default:
			}
		}
	}

	return nil
}

func (m *Manager) Get() *Snapshot {
	return m.snapshot.Load()
}

func (m *Manager) Subscribe(ctx context.Context, name string, fn func(context.Context, *Snapshot) error, mode ApplyMode) {
	m.mu.Lock()
	defer m.mu.Unlock()

	s := &subscriber{
		name: name,
		fn:   fn,
		mode: mode,
	}
	if mode == ApplyAsync {
		s.ch = make(chan *Snapshot, 1)
	}

	m.subs = append(m.subs, s)

	if mode == ApplyAsync {
		go m.asyncSubLoop(ctx, s)
	}
}

func (m *Manager) asyncSubLoop(ctx context.Context, s *subscriber) {
	var last *Snapshot
	for snap := range s.ch {
		if snap == last {
			continue
		}

		last = snap
		if err := s.fn(ctx, snap); err != nil {
			if m.logger != nil {
				m.logger.Errorw("async subscriber failed", "name", s.name, "error", err)
			}
		}
	}
}

func (m *Manager) Stop() {
	if m.stopped.Swap(true) {
		return
	}
}
