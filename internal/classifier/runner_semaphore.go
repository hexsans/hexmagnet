package classifier

import (
	"context"

	"github.com/hexsans/hexmagnet/internal/model"
)

type runnerSemaphore struct {
	runner    Runner
	semaphore chan struct{}
}

func (r runnerSemaphore) Run(
	ctx context.Context,
	workflow string,
	flags Flags,
	t model.Torrent,
) (ClassificationResult, error) {
	select {
	case <-ctx.Done():
		return ClassificationResult{}, ctx.Err()
	case r.semaphore <- struct{}{}:
	}

	defer func() { <-r.semaphore }()

	return r.runner.Run(ctx, workflow, flags, t)
}
