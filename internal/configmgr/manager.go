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
	"github.com/hexsans/hexmagnet/internal/retryqueue"
	"github.com/hexsans/hexmagnet/internal/servercfg"
	"github.com/hexsans/hexmagnet/internal/torznab"
	"github.com/hexsans/hexmagnet/internal/webhook"
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
	Torznab      torznab.Config
	Webhooks     webhook.Config
	RetryQueue   retryqueue.Config
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
	snapshot   atomic.Pointer[Snapshot]
	subs       []*subscriber
	logger     *zap.SugaredLogger
	mu         sync.Mutex
	stopped    atomic.Bool
	stopCtx    context.Context
	stopCancel context.CancelFunc
}

func NewManager(initial *Snapshot, logger *zap.SugaredLogger) *Manager {
	if initial == nil {
		initial = &Snapshot{}
	}

	stopCtx, stopCancel := context.WithCancel(context.Background())

	m := &Manager{
		logger:     logger,
		stopCtx:    stopCtx,
		stopCancel: stopCancel,
	}
	m.snapshot.Store(initial)

	return m
}

func (m *Manager) Apply(ctx context.Context, snap *Snapshot) error {
	m.mu.Lock()
	subs := make([]*subscriber, len(m.subs))
	copy(subs, m.subs)
	m.mu.Unlock()

	// Sync subscribers run first: if any fail, the snapshot is left untouched
	// so the service keeps running on the old values. Persisting the config
	// file is the caller's responsibility.
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

// SetLogger attaches the application logger after construction. It exists to
// break the dependency cycle between the logger and the config manager (the
// logging subsystem subscribes to config updates). It must be called before
// the first Apply.
func (m *Manager) SetLogger(logger *zap.SugaredLogger) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger = logger
}

func (m *Manager) Subscribe(name string, fn func(context.Context, *Snapshot) error, mode ApplyMode) {
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
		go m.asyncSubLoop(s)
	}
}

func (m *Manager) asyncSubLoop(s *subscriber) {
	var last *Snapshot

	for {
		select {
		case <-m.stopCtx.Done():
			return
		case snap := <-s.ch:
			if snap == last {
				continue
			}

			last = snap
			if err := s.fn(m.stopCtx, snap); err != nil {
				if m.logger != nil {
					m.logger.Errorw("async subscriber failed", "name", s.name, "error", err)
				}
			}
		}
	}
}

// Stop cancels the manager's lifetime context, terminating all async
// subscriber goroutines. It is idempotent.
func (m *Manager) Stop() {
	if m.stopped.Swap(true) {
		return
	}

	m.stopCancel()
}
