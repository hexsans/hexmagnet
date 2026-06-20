package responder

import (
	"context"
	"net/netip"
	"sync"
	"time"

	"github.com/hexsans/hexmagnet/internal/concurrency"
	"github.com/hexsans/hexmagnet/internal/protocol/dht"
	"golang.org/x/time/rate"
)

// responderLimiter applies both overall and per-IP rate limiting
type responderLimiter struct {
	responder Responder
	limiter   Limiter
}

func (r responderLimiter) Respond(ctx context.Context, msg dht.RecvMsg) (ret dht.Return, err error) {
	if !r.limiter.Allow(msg.From.Addr()) {
		return dht.Return{}, ErrTooManyRequests
	}

	return r.responder.Respond(ctx, msg)
}

func (r responderLimiter) SetGlobalRateLimit(n int) {
	r.limiter.SetGlobalRateLimit(n)
}

func (r responderLimiter) SetPerIPRateLimit(n int) {
	r.limiter.SetPerIPRateLimit(n)
}

type Limiter interface {
	Allow(addr netip.Addr) bool
	SetGlobalRateLimit(n int)
	SetPerIPRateLimit(n int)
}

type limiter struct {
	keyedLimiter concurrency.KeyedLimiter
	mu           sync.RWMutex
	limiter      *rate.Limiter
}

func NewLimiter(
	overallRate rate.Limit,
	overallBurst int,
	perIPRate rate.Limit,
	perIPBurst int,
	perIPSize int,
	perIPTTL time.Duration,
) Limiter {
	l := &limiter{
		limiter: rate.NewLimiter(overallRate, overallBurst),
	}
	l.keyedLimiter = concurrency.NewKeyedLimiter(perIPRate, perIPBurst, perIPSize, perIPTTL)

	return l
}

func (l *limiter) Allow(addr netip.Addr) bool {
	l.mu.RLock()
	kl := l.keyedLimiter
	l.mu.RUnlock()

	return kl.Allow(addr.String()) && l.limiter.Allow()
}

func (l *limiter) SetGlobalRateLimit(n int) {
	if n <= 0 {
		l.limiter.SetLimit(rate.Inf)
		l.limiter.SetBurst(20)
	} else {
		l.limiter.SetLimit(rate.Limit(n))
		l.limiter.SetBurst(n)
	}
}

func (l *limiter) SetPerIPRateLimit(n int) {
	var rl rate.Limit

	burst := 10

	if n > 0 {
		rl = rate.Limit(n)
		burst = n
	}

	l.mu.Lock()
	l.keyedLimiter = concurrency.NewKeyedLimiter(rl, burst, 1000, time.Second*20)
	l.mu.Unlock()
}
