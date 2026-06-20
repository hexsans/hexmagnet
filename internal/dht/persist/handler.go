package persist

import (
	"context"

	"github.com/hexsans/hexmagnet/internal/dht"
	"go.uber.org/zap"
)

type Result struct {
	InfoHash string
	Scrape   bool
}

type Handler interface {
	HandlePersist(ctx context.Context, msg dht.MetaInfoMessage) (Result, error)
}

type handler struct {
	logger *zap.SugaredLogger
}

func New(logger *zap.SugaredLogger) Handler {
	return &handler{logger: logger.Named("persist")}
}

func (h *handler) HandlePersist(_ context.Context, msg dht.MetaInfoMessage) (Result, error) {
	_, err := dht.ParseInfoHash(h.logger, msg.InfoHash, "persist")
	if err != nil {
		return Result{}, err
	}

	return Result{
		InfoHash: msg.InfoHash,
		Scrape:   true,
	}, nil
}
