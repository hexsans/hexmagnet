package indexer

import (
	"errors"
	"testing"

	"github.com/hexsans/hexmagnet/internal/database/db"
	"github.com/hexsans/hexmagnet/internal/queue"
	"github.com/hexsans/hexmagnet/internal/utils"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestDefaultSearchConfig_ReturnsConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultSearchConfig()
	assert.Equal(t, "postgresql", cfg.Backend)
}

func TestNew_ReturnsResult(t *testing.T) {
	t.Parallel()

	r := queue.NewRuntime(queue.Config{Backend: "memory"}, zap.NewNop().Sugar())

	p := Params{
		SearchCfg:     DefaultSearchConfig(),
		Queries:       utils.NewLazy(func() (*db.Queries, error) { return nil, errors.New("unused in test") }),
		ConsumerMaker: r.DynamicConsumerMaker(),
		Logger:        zap.NewNop().Sugar(),
		Runtime:       r,
	}

	result := New(p)
	assert.NotNil(t, result.Worker)
	assert.Equal(t, "message_queue_enrich_indexer", result.Worker.Key())
}
