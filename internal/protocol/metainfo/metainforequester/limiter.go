package metainforequester

import (
	"context"
	"net/netip"

	"github.com/hexsans/hexmagnet/internal/concurrency"
	"github.com/hexsans/hexmagnet/internal/protocol"
)

type requestLimiter struct {
	requester Requester
	limiter   concurrency.KeyedLimiter
}

func (r requestLimiter) Request(ctx context.Context, infoHash protocol.ID, node netip.AddrPort) (Response, error) {
	if limitErr := r.limiter.Wait(ctx, node.Addr().String()); limitErr != nil {
		return Response{}, limitErr
	}

	return r.requester.Request(ctx, infoHash, node)
}
