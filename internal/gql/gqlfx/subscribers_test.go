package gqlfx

import (
	"context"
	"net/netip"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hexsans/hexmagnet/internal/classifier"
	"github.com/hexsans/hexmagnet/internal/configmgr"
	"github.com/hexsans/hexmagnet/internal/protocol/dht"
	"github.com/hexsans/hexmagnet/internal/protocol/metainfo/metainforequester"
	"github.com/hexsans/hexmagnet/internal/tmdb"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"golang.org/x/time/rate"
)

type fakeResponderLimiter struct {
	mu     sync.Mutex
	global int
	perIP  int
}

func (*fakeResponderLimiter) Allow(_ netip.Addr) bool { return true }
func (f *fakeResponderLimiter) SetGlobalRateLimit(n int) {
	f.mu.Lock()
	f.global = n
	f.mu.Unlock()
}

func (f *fakeResponderLimiter) SetPerIPRateLimit(n int) {
	f.mu.Lock()
	f.perIP = n
	f.mu.Unlock()
}

type fakeTmdbUpdater struct {
	mu   sync.Mutex
	cfgs []tmdb.Config
}

func (f *fakeTmdbUpdater) UpdateConfig(cfg tmdb.Config) {
	f.mu.Lock()
	f.cfgs = append(f.cfgs, cfg)
	f.mu.Unlock()
}

func waitFor(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}

		time.Sleep(10 * time.Millisecond)
	}

	t.Fatal("condition not met within timeout")
}

// TestGlobalSubscribers_RevertToStartupValueIsApplied guards against the
// stale-dedup bug where subscribers compared the incoming snapshot against the
// startup config captured in the closure: after changing a value and reverting
// it to the startup value, the runtime kept the stale value until restart.
func TestGlobalSubscribers_RevertToStartupValueIsApplied(t *testing.T) {
	t.Parallel()

	logger := zaptest.NewLogger(t).Sugar()

	cm := configmgr.NewManager(nil, "", nil, logger)
	defer cm.Stop()

	limiter := rate.NewLimiter(rate.Limit(100), 100)
	enabled := &atomic.Bool{}
	enabled.Store(true)

	rl := &fakeResponderLimiter{}
	updater := &fakeTmdbUpdater{}

	startupDHT := dht.Config{
		Port: 3334,
		Responder: dht.ResponderConfig{
			Enabled:         true,
			GlobalRateLimit: 100,
			PerIPRateLimit:  100,
		},
	}
	startupRequester := metainforequester.Config{RequestLimit: 100}

	p := GlobalSubscriberParams{
		ConfigManager:        cm,
		RequestLimiter:       limiter,
		ResponderEnabled:     enabled,
		ResponderRateLimiter: rl,
		TmdbUpdater:          updater,
		DHTRequesterCfg:      startupRequester,
		DhtCfg:               startupDHT,
	}
	registerGlobalSubscribers(p)

	changed := &configmgr.Snapshot{
		DHTRequester: metainforequester.Config{RequestLimit: 50},
		DHT: dht.Config{
			Port:      3334,
			Responder: dht.ResponderConfig{Enabled: false, GlobalRateLimit: 10, PerIPRateLimit: 5},
		},
		Classifier: classifier.Config{Tmdb: tmdb.Config{Enabled: false, AccessToken: "a", RateLimit: 3}},
	}
	require.NoError(t, cm.Apply(context.Background(), changed))

	waitFor(t, 2*time.Second, func() bool {
		return !enabled.Load() && limiter.Limit() == rate.Limit(50)
	})

	// Revert every value back to the startup value.
	reverted := &configmgr.Snapshot{
		DHTRequester: startupRequester,
		DHT:          startupDHT,
		Classifier:   classifier.Config{Tmdb: tmdb.Config{Enabled: true, AccessToken: "b", RateLimit: 20}},
	}
	require.NoError(t, cm.Apply(context.Background(), reverted))

	waitFor(t, 2*time.Second, func() bool {
		rl.mu.Lock()
		ok := enabled.Load() &&
			limiter.Limit() == rate.Limit(100) &&
			rl.global == 100 && rl.perIP == 100
		rl.mu.Unlock()

		return ok
	})

	updater.mu.Lock()
	defer updater.mu.Unlock()

	require.Len(t, updater.cfgs, 2)
	require.Equal(t, tmdb.Config{Enabled: false, AccessToken: "a", RateLimit: 3}, updater.cfgs[0])
	require.Equal(t, tmdb.Config{Enabled: true, AccessToken: "b", RateLimit: 20}, updater.cfgs[1])
}
