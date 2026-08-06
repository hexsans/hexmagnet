package httpserver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hexsans/hexmagnet/internal/servercfg"
	"github.com/hexsans/hexmagnet/internal/worker"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In
	Config       Config
	Options      []Option `group:"http_server_options"`
	Logger       *zap.Logger
	ServerConfig servercfg.Config
}

type Result struct {
	fx.Out
	Worker worker.Worker `group:"workers"`
}

func New(p Params) Result {
	var s *http.Server

	return Result{
		Worker: worker.NewWorker(
			"http_server",
			fx.Hook{
				OnStart: func(ctx context.Context) error {
					gin.SetMode(p.Config.GinMode)

					g := gin.New()
					g.Use(Ginzap(p.Logger.Named("gin"), time.RFC3339, true), gin.Recovery())

					options, optionsErr := resolveOptions(p.Config.Options, p.Options)
					if optionsErr != nil {
						return optionsErr
					}

					for _, o := range options {
						if buildErr := o.Apply(g); buildErr != nil {
							return buildErr
						}
					}

					addr := fmt.Sprintf("%s:%d", p.ServerConfig.IP, p.ServerConfig.Port)

					s = &http.Server{
						Addr:         addr,
						Handler:      g.Handler(),
						ReadTimeout:  p.Config.ReadTimeout,
						WriteTimeout: p.Config.WriteTimeout,
						IdleTimeout:  p.Config.IdleTimeout,
					}

					ln, listenErr := (&net.ListenConfig{}).Listen(ctx, "tcp", s.Addr)
					if listenErr != nil {
						return listenErr
					}

					p.Logger.Info("http server listening", zap.String("addr", s.Addr))

					go func() {
						serveErr := s.Serve(ln)
						if !errors.Is(serveErr, http.ErrServerClosed) {
							p.Logger.Error("http server exited unexpectedly",
								zap.Error(serveErr),
							)
						}
					}()

					return nil
				},
				OnStop: func(ctx context.Context) error {
					if s == nil {
						return nil
					}

					return s.Shutdown(ctx)
				},
			},
		),
	}
}

func resolveOptions(param []string, options []Option) ([]Option, error) {
	paramMap := make(map[string]struct{})
	for _, p := range param {
		paramMap[p] = struct{}{}
	}

	enabledOptions := make([]Option, 0, len(options))

	foundMap := make(map[string]struct{}, len(options))
	for _, o := range options {
		if _, ok := foundMap[o.Key()]; ok {
			return nil, fmt.Errorf("duplicate http server option: '%s'", o.Key())
		}

		foundMap[o.Key()] = struct{}{}

		enabled := false
		if _, ok := paramMap["*"]; ok {
			enabled = true
		} else if _, ok := paramMap[o.Key()]; ok {
			enabled = true
		}

		if enabled {
			enabledOptions = append(enabledOptions, o)
		}
	}

	for p := range paramMap {
		if _, ok := foundMap[p]; !ok && p != "*" {
			return nil, fmt.Errorf("unknown http server option: '%s'", p)
		}
	}

	sort.Slice(enabledOptions, func(i, j int) bool {
		return enabledOptions[i].Key() < enabledOptions[j].Key()
	})

	return enabledOptions, nil
}

type Option interface {
	Key() string
	Apply(engine *gin.Engine) error
}
