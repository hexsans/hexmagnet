package httpserver

import (
	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gin-gonic/gin"
	"github.com/hexsans/hexmagnet/internal/httpserver"
	"github.com/hexsans/hexmagnet/internal/utils"
	"github.com/vektah/gqlparser/v2/ast"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In
	Schema utils.Lazy[graphql.ExecutableSchema]
	Logger *zap.SugaredLogger
}

type Result struct {
	fx.Out
	Option httpserver.Option `group:"http_server_options"`
}

func New(p Params) Result {
	return Result{
		Option: &builder{
			schema: p.Schema,
			logger: p.Logger.Named("graphql"),
		},
	}
}

type builder struct {
	schema utils.Lazy[graphql.ExecutableSchema]
	logger *zap.SugaredLogger
}

func (builder) Key() string {
	return "graphql"
}

func (b builder) Apply(e *gin.Engine) error {
	schema, err := b.schema.Get()
	if err != nil {
		b.logger.Errorw("failed to get graphql schema", "error", err)
		return err
	}

	gql := newServer(schema)

	e.POST("/graphql", func(c *gin.Context) {
		gql.ServeHTTP(c.Writer, c.Request)
	})

	pg := playground.Handler("GraphQL playground", "/graphql")

	e.GET("/graphql", func(c *gin.Context) {
		pg.ServeHTTP(c.Writer, c.Request)
	})

	return nil
}

func newServer(es graphql.ExecutableSchema) *handler.Server {
	srv := handler.New(es)

	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.MultipartForm{})

	srv.SetQueryCache(lru.New[*ast.QueryDocument](100))

	srv.Use(extension.Introspection{})
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New[string](100),
	})

	return srv
}
