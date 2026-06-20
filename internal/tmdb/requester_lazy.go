package tmdb

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/hexsans/hexmagnet/internal/concurrency"
	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"
	"golang.org/x/time/rate"
)

// requesterLazy defers instantiation of the requester (and possible failure) until the first request is made,
// avoiding failure when the TMDB client is not needed.
type requesterLazy struct {
	mu                sync.Mutex
	config            Config
	logger            *zap.SugaredLogger
	err               error
	requester         Requester
	accessTokenHolder *AccessTokenHolder
	lastToken         string
}

// UpdateConfig applies a runtime config change. The requester is rebuilt on
// the next request so token, rate limit and enabled changes take effect
// immediately.
func (r *requesterLazy) UpdateConfig(cfg Config) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.config == cfg {
		return
	}

	r.config = cfg
	r.accessTokenHolder.Set(cfg.AccessToken)
	r.requester = nil
	r.err = nil
	r.lastToken = ""
}

func (r *requesterLazy) Request(
	ctx context.Context,
	path string,
	queryParams map[string]string,
	result any,
) (*resty.Response, error) {
	r.mu.Lock()

	currentToken := r.accessTokenHolder.Get()
	if currentToken == "" {
		r.mu.Unlock()
		return requesterNoop{}.Request(ctx, path, queryParams, result)
	}

	if r.requester == nil || currentToken != r.lastToken {
		cfg := r.config
		cfg.AccessToken = currentToken
		r.requester, r.err = newRequester(ctx, cfg, r.logger)
		r.lastToken = currentToken
	}

	req := r.requester
	err := r.err
	r.mu.Unlock()

	if err != nil {
		return nil, err
	}

	return req.Request(ctx, path, queryParams, result)
}

type requesterNoop struct{}

func (requesterNoop) Request(_ context.Context, _ string, _ map[string]string, _ any) (*resty.Response, error) {
	return nil, nil //nolint:nilnil // noop: disabled TMDB client returns no response and no error
}

func newRequester(ctx context.Context, config Config, logger *zap.SugaredLogger) (Requester, error) {
	if !config.Enabled {
		return nil, errors.New("TMDB is disabled")
	}

	if config.AccessToken == "" {
		return requesterNoop{}, nil
	}

	r := requesterLogger{
		requester: requesterFailFast{
			requester: requesterSemaphore{
				requester: requesterLimiter{
					requester: requester{
						resty: resty.New().
							SetTransport(&http.Transport{
								MaxIdleConns:        10,
								MaxIdleConnsPerHost: 10,
								MaxConnsPerHost:     20,
								IdleConnTimeout:     90 * time.Second,
							}).
							SetBaseURL("https://api.themoviedb.org/3").
							SetAuthToken(config.AccessToken).
							SetRetryCount(3).
							SetRetryWaitTime(2 * time.Second).
							SetRetryMaxWaitTime(20 * time.Second).
							SetTimeout(10 * time.Second).
							EnableTrace().
							SetLogger(logger),
					},
					limiter: rate.NewLimiter(rate.Limit(config.RateLimit), config.RateLimit),
				},
				semaphore: semaphore.NewWeighted(2),
			},
			isUnauthorized: &concurrency.AtomicValue[bool]{},
		},
		logger: logger,
	}

	err := client{r}.ValidateAccessToken(ctx)
	if err != nil {
		return r, fmt.Errorf("TMDB access token validation failed: %w", err)
	}

	return r, nil
}
