package httpserver

import (
	"context"
	"testing"
	"time"

	"github.com/hexsans/hexmagnet/internal/servercfg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func TestNewModule(t *testing.T) {
	t.Parallel()

	app := fx.New(
		fx.Supply(zap.NewNop()),
		fx.Supply(servercfg.NewDefaultConfig()),
		NewModule(),
		fx.Invoke(func(Config) {}),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	require.NoError(t, app.Start(ctx))
	require.NoError(t, app.Stop(ctx))
}

func TestNewModule_OptionNotNil(t *testing.T) {
	t.Parallel()

	mod := NewModule()
	assert.NotNil(t, mod)
}
