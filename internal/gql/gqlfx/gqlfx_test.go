package gqlfx

import (
	"testing"

	"github.com/hexsans/hexmagnet/internal/gql"
	"github.com/hexsans/hexmagnet/internal/gql/resolvers"
	"github.com/hexsans/hexmagnet/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	opt := New()
	assert.NotNil(t, opt)
}

func TestNewGqlConfig(t *testing.T) {
	t.Parallel()

	r := &resolvers.Resolver{}
	lazyR := utils.NewLazy(func() (*resolvers.Resolver, error) { return r, nil })
	p := GqlConfigParams{ResolverRoot: lazyR}
	result := NewGqlConfig(p)
	cfg, err := result.Get()
	require.NoError(t, err)
	assert.IsType(t, gql.Config{}, cfg)
}

func TestNewGqlConfig_GetError(t *testing.T) {
	t.Parallel()

	expectedErr := assert.AnError
	lazyR := utils.NewLazy(func() (*resolvers.Resolver, error) { return nil, expectedErr })
	p := GqlConfigParams{ResolverRoot: lazyR}
	result := NewGqlConfig(p)
	_, err := result.Get()
	require.Error(t, err)
	assert.ErrorIs(t, err, expectedErr)
}
