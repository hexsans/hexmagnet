package server

import (
	"context"
	"net/netip"

	"github.com/hexsans/hexmagnet/internal/protocol/dht"
	"golang.org/x/time/rate"
)

type globalRequestRateLimiter struct {
	server  Server
	limiter *rate.Limiter
}

func (s globalRequestRateLimiter) Query(ctx context.Context, addr netip.AddrPort, q string, args dht.MsgArgs) (dht.RecvMsg, error) {
	if err := s.limiter.Wait(ctx); err != nil {
		return dht.RecvMsg{}, err
	}

	return s.server.Query(ctx, addr, q, args)
}

func (s globalRequestRateLimiter) start() error {
	return s.server.start()
}

func (s globalRequestRateLimiter) stop() {
	s.server.stop()
}
