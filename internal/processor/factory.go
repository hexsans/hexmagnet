package processor

import (
	"context"
	"runtime/debug"
	"time"

	"github.com/hexsans/hexmagnet/internal/blocking"
	"github.com/hexsans/hexmagnet/internal/classifier"
	"github.com/hexsans/hexmagnet/internal/configmgr"
	"github.com/hexsans/hexmagnet/internal/database/db"
	"github.com/hexsans/hexmagnet/internal/queue"
	"github.com/hexsans/hexmagnet/internal/retryqueue"
	dbsearch "github.com/hexsans/hexmagnet/internal/search"
	"github.com/hexsans/hexmagnet/internal/utils"
	"github.com/hexsans/hexmagnet/internal/webhook"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In
	ClassifierConfig classifier.Config
	SearchRuntime    *dbsearch.Runtime
	Workflow         utils.Lazy[classifier.Runner]
	Queries          utils.Lazy[*db.Queries]
	BlockingManager  utils.Lazy[blocking.Manager]
	Producer         queue.Producer
	ConfigManager    *configmgr.Manager                                 `optional:"true"`
	RebuildRunner    func(classifier.Config) (classifier.Runner, error) `optional:"true"`
	WebhookPublisher *webhook.Publisher                                 `optional:"true"`
	RetryQueue       *retryqueue.Queue                                  `optional:"true"`
	Logger           *zap.SugaredLogger
}

type Result struct {
	fx.Out
	Processor utils.Lazy[Processor]
}

func New(p Params) Result {
	return Result{
		Processor: utils.NewLazy(func() (Processor, error) {
			q, err := p.Queries.Get()
			if err != nil {
				return nil, err
			}

			bm, err := p.BlockingManager.Get()
			if err != nil {
				return nil, err
			}

			w, err := p.Workflow.Get()
			if err != nil {
				return nil, err
			}

			proc := &processor{
				searchRuntime:   p.SearchRuntime,
				queries:         q,
				blockingManager: bm,
				runner:          w,
				kafkaProducer:   p.Producer,
				webhook:         p.WebhookPublisher,
				retryQueue:      p.RetryQueue,
				defaultWorkflow: "default",
				logger:          p.Logger,
			}

			initialFilter, err := compileFilterState(p.ClassifierConfig.TorrentFilter)
			if err != nil {
				p.Logger.Warnw("invalid torrent filter config from file, using off",
					"error", err,
				)

				initialFilter, _ = compileFilterState(classifier.TorrentFilterConfig{Mode: classifier.TorrentFilterOff})
			}

			proc.filter.Store(initialFilter)

			if p.ConfigManager != nil && p.RebuildRunner != nil {
				p.ConfigManager.Subscribe(context.Background(), "classifier",
					func(_ context.Context, snap *configmgr.Snapshot) error {
						if err := proc.UpdateTorrentFilter(snap.Classifier.TorrentFilter); err != nil {
							p.Logger.Warnw("failed to update torrent filter", "error", err)
							return nil
						}

						newRunner, err := p.RebuildRunner(snap.Classifier)
						if err != nil {
							p.Logger.Warnw("failed to rebuild classifier runner", "error", err)
							return nil
						}

						if newRunner != nil {
							proc.SwapRunner(newRunner)
						}

						return nil
					}, configmgr.ApplyAsync)
			}

			go func() {
				defer func() {
					if r := recover(); r != nil {
						proc.logger.Errorw("classification cache cleanup panicked",
							"panic", r,
							"stack", string(debug.Stack()),
						)
					}
				}()

				ticker := time.NewTicker(1 * time.Minute)
				defer ticker.Stop()

				for range ticker.C {
					proc.recentlyClassified.Range(func(key, value interface{}) bool {
						t, ok := value.(time.Time)
						if !ok {
							proc.recentlyClassified.Delete(key)
							return true
						}

						if time.Since(t) > 3*time.Minute {
							proc.recentlyClassified.Delete(key)
						}

						return true
					})
				}
			}()

			return proc, nil
		}),
	}
}
