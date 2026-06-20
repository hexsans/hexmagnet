package tmdb

import (
	"context"

	"go.uber.org/zap"
)

// ValidateAccessToken checks that the given config can reach the TMDB API
// with a valid access token. Returns nil when TMDB is disabled or the token
// is empty, as there is nothing to validate. No state is mutated.
func ValidateAccessToken(ctx context.Context, config Config, logger *zap.SugaredLogger) error {
	if !config.Enabled || config.AccessToken == "" {
		return nil
	}

	_, err := newRequester(ctx, config, logger)

	return err
}
