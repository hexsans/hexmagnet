package metainforequester

import (
	"context"
	"net/netip"

	"github.com/hexsans/hexmagnet/internal/protocol"
	"golang.org/x/time/rate"
)

type requestRateLimiter struct {
	requester Requester
	limiter   *rate.Limiter
}

func (r *requestRateLimiter) Request(ctx context.Context, infoHash protocol.ID, addr netip.AddrPort) (Response, error) {
	if err := r.limiter.Wait(ctx); err != nil {
		return Response{}, err
	}

	return r.requester.Request(ctx, infoHash, addr)
}
