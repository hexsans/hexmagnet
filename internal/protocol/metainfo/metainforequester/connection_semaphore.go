package metainforequester

import (
	"context"
	"net/netip"

	"github.com/hexsans/hexmagnet/internal/protocol"
	"golang.org/x/sync/semaphore"
)

type connectionSemaphore struct {
	requester Requester
	sem       *semaphore.Weighted
}

func (r *connectionSemaphore) Request(ctx context.Context, infoHash protocol.ID, addr netip.AddrPort) (Response, error) {
	if err := r.sem.Acquire(ctx, 1); err != nil {
		return Response{}, err
	}
	defer r.sem.Release(1)

	return r.requester.Request(ctx, infoHash, addr)
}
