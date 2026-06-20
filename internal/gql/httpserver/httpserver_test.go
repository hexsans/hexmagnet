package httpserver

import (
	"context"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/gin-gonic/gin"
	"github.com/hexsans/hexmagnet/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"
	"go.uber.org/zap"
)

func TestNew(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	sugar := logger.Sugar()

	schema := utils.NewLazy(func() (graphql.ExecutableSchema, error) {
		return nil, assert.AnError
	})
	p := Params{Schema: schema, Logger: sugar}
	result := New(p)
	assert.NotNil(t, result.Option)
}

func TestBuilder_Key(t *testing.T) {
	t.Parallel()

	b := builder{}
	assert.Equal(t, "graphql", b.Key())
}

func TestBuilder_Apply_Error(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	sugar := logger.Sugar()

	schema := utils.NewLazy(func() (graphql.ExecutableSchema, error) {
		return nil, assert.AnError
	})
	b := builder{schema: schema, logger: sugar}
	engine := gin.New()
	err = b.Apply(engine)
	require.Error(t, err)
	assert.ErrorIs(t, err, assert.AnError)
}

func TestBuilder_Apply_Success(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	sugar := logger.Sugar()

	schema := utils.NewLazy(func() (graphql.ExecutableSchema, error) {
		return &graphql.ExecutableSchemaMock{
			ComplexityFunc: func(_ context.Context, _, _ string, _ int, _ map[string]any) (int, bool) { return 0, false },
			ExecFunc:       func(_ context.Context) graphql.ResponseHandler { return nil },
			SchemaFunc:     func() *ast.Schema { return nil },
		}, nil
	})
	b := builder{schema: schema, logger: sugar}
	engine := gin.New()
	err = b.Apply(engine)
	require.NoError(t, err)
}
