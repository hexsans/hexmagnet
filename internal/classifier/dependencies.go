package classifier

import (
	"github.com/hexsans/hexmagnet/internal/tmdb"
	"go.uber.org/zap"
)

type dependencies struct {
	search     LocalSearch
	tmdbClient tmdb.Client
	logger     *zap.SugaredLogger
	llmClient  *Client
	llmEnabled bool
}
