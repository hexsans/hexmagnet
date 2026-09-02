package processor

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hexsans/hexmagnet/internal/blocking"
	"github.com/hexsans/hexmagnet/internal/classifier"
	"github.com/hexsans/hexmagnet/internal/database/db"
	"github.com/hexsans/hexmagnet/internal/model"
	"github.com/hexsans/hexmagnet/internal/protocol"
	"github.com/hexsans/hexmagnet/internal/queue"
	"github.com/hexsans/hexmagnet/internal/queue/kafka"
	dbsearch "github.com/hexsans/hexmagnet/internal/search"
	"github.com/hexsans/hexmagnet/internal/webhook"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
)

type Processor interface {
	Process(ctx context.Context, params MessageParams) error
	UpdateTorrentFilter(cfg classifier.TorrentFilterConfig) error
	SwapRunner(runner classifier.Runner)
}

type filterState struct {
	mode             classifier.TorrentFilterMode
	titlePatterns    []*regexp.Regexp
	filenamePatterns []*regexp.Regexp
}

func compileFilterState(cfg classifier.TorrentFilterConfig) (*filterState, error) {
	fs := &filterState{mode: cfg.Mode}
	for _, pattern := range cfg.TitlePatterns {
		re, err := regexp.Compile("(?i)" + pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid title pattern %q: %w", pattern, err)
		}

		fs.titlePatterns = append(fs.titlePatterns, re)
	}

	for _, pattern := range cfg.FilenamePatterns {
		re, err := regexp.Compile("(?i)" + pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid filename pattern %q: %w", pattern, err)
		}

		fs.filenamePatterns = append(fs.filenamePatterns, re)
	}

	return fs, nil
}

type processor struct {
	defaultWorkflow    string
	searchRuntime      *dbsearch.Runtime
	runner             classifier.Runner
	runnerAtomic       atomic.Pointer[classifierRunnerHolder]
	queries            *db.Queries
	blockingManager    blocking.Manager
	kafkaProducer      queue.Producer
	webhook            *webhook.Publisher
	logger             *zap.SugaredLogger
	sf                 singleflight.Group
	recentlyClassified sync.Map
	filter             atomic.Pointer[filterState]
}

type classifierRunnerHolder struct {
	runner classifier.Runner
}

func (c *processor) wasRecentlyClassified(ih protocol.ID) bool {
	v, ok := c.recentlyClassified.Load(ih)
	if !ok {
		return false
	}

	t, ok := v.(time.Time)
	if !ok {
		return false
	}

	return time.Since(t) < 3*time.Minute
}

func (c *processor) markRecentlyClassified(ih protocol.ID) {
	c.recentlyClassified.Store(ih, time.Now())
}

type MissingHashesError struct {
	InfoHashes []protocol.ID
}

func (e MissingHashesError) Error() string {
	return fmt.Sprintf("missing %d info hashes", len(e.InfoHashes))
}

func (c *processor) Process(ctx context.Context, params MessageParams) error {
	workflowName := c.defaultWorkflow

	searchResult, err := c.fetchAndJoinData(ctx, params)
	if err != nil {
		return err
	}

	var (
		mtx  sync.Mutex
		wg   sync.WaitGroup
		errs []error
	)

	tcs := make([]model.Torrent, 0, len(searchResult.Torrents))

	failedHashes := make([]protocol.ID, 0, len(searchResult.MissingInfoHashes))
	for _, h := range searchResult.MissingInfoHashes {
		failedHashes = append(failedHashes, db.ToProtocolID(h))
	}

	if len(failedHashes) > 0 {
		errs = append(errs, MissingHashesError{InfoHashes: failedHashes})
	}

	for _, torrent := range searchResult.Torrents {
		wg.Add(1)

		go func(torrent model.Torrent) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					c.logger.Errorw("classification goroutine panicked",
						"info_hash", torrent.InfoHash.String(),
						"panic", r,
						"stack", string(debug.Stack()),
					)
				}
			}()

			if c.wasRecentlyClassified(torrent.InfoHash) {
				return
			}

			if c.filteredByTorrentFilter(torrent) {
				hash := torrent.InfoHash
				if err := c.queries.DeleteTorrent(ctx, db.FromProtocolID(hash)); err != nil {
					c.logger.Errorw("failed to delete filtered torrent", "info_hash", hash.String(), "error", err)
				}

				if err := c.blockingManager.Block(ctx, hash, "torrent filter"); err != nil {
					c.logger.Errorw("failed to block filtered torrent", "info_hash", hash.String(), "error", err)
				}

				c.kafkaProducer.Produce(kafka.TopicDeleteTorrent, hash.String(), hash.String())

				return
			}

			type classifyResult struct {
				result classifier.ClassificationResult
				err    error
			}

			key := torrent.InfoHash.String()
			v, _, shared := c.sf.Do(key, func() (interface{}, error) {
				c.markRecentlyClassified(torrent.InfoHash)

				rn := c.runner
				if h := c.runnerAtomic.Load(); h != nil {
					rn = h.runner
				}

				cl, classifyErr := rn.Run(context.WithoutCancel(ctx), workflowName, nil, torrent)

				return classifyResult{result: cl, err: classifyErr}, nil
			})

			cr, ok := v.(classifyResult)
			if !ok {
				mtx.Lock()

				errs = append(errs, fmt.Errorf("unexpected type from singleflight: %T", v))
				failedHashes = append(failedHashes, torrent.InfoHash)
				mtx.Unlock()

				return
			}

			cl, classifyErr := cr.result, cr.err

			if shared {
				c.logger.Debugw("deduplicated classification", "info_hash", key)
			}

			mtx.Lock()
			if classifyErr != nil {
				c.sf.Forget(key)
				c.logger.Warnw("classification failed for torrent",
					"info_hash", torrent.InfoHash.String(),
					"workflow", workflowName,
					"error", classifyErr,
				)
				failedHashes = append(failedHashes, torrent.InfoHash)
				errs = append(errs, classifyErr)
			} else {
				torrentContent := newTorrentContent(torrent, cl)

				if params.ContentType != "" {
					torrentContent.ContentType = model.NewNullContentType(params.ContentType)
				}

				tcs = append(tcs, torrentContent)
			}
			mtx.Unlock()
		}(torrent)
	}

	wg.Wait()

	return c.handleClassified(ctx, tcs, failedHashes, errs, params)
}

func (c *processor) fetchAndJoinData(ctx context.Context, params MessageParams) (dbsearch.TorrentsWithMissingInfoHashesResult, error) {
	infoHashes := make([]string, len(params.InfoHashes))
	for i, ih := range params.InfoHashes {
		infoHashes[i] = db.FromProtocolID(ih)
	}

	searchResult, err := c.searchRuntime.Search.Get().TorrentsWithMissingInfoHashes(ctx, dbsearch.TorrentsWithMissingInfoHashesParams{
		InfoHashes: infoHashes,
	})
	if err != nil {
		return dbsearch.TorrentsWithMissingInfoHashesResult{}, err
	}

	return searchResult, nil
}

func (c *processor) filteredByTorrentFilter(torrent model.Torrent) bool {
	fs := c.filter.Load()
	if fs.mode == classifier.TorrentFilterOff {
		return false
	}

	match := false

	for _, p := range fs.titlePatterns {
		if p.MatchString(torrent.Name) {
			match = true
			break
		}
	}

	if len(fs.filenamePatterns) > 0 && !match {
		for _, f := range torrent.Files {
			path := strings.Join(f.PathParts, "/")
			for _, p := range fs.filenamePatterns {
				if p.MatchString(path) {
					match = true
					break
				}
			}

			if match {
				break
			}
		}
	}

	if fs.mode == classifier.TorrentFilterDiscard && match {
		c.logger.Debugw("torrent discarded by torrent filter",
			"name", torrent.Name,
			"info_hash", torrent.InfoHash.String(),
		)

		return true
	}

	if fs.mode == classifier.TorrentFilterProcess && !match {
		c.logger.Debugw("torrent skipped by torrent filter (not matched)",
			"name", torrent.Name,
			"info_hash", torrent.InfoHash.String(),
		)

		return true
	}

	return false
}

func (c *processor) handleClassified(
	ctx context.Context,
	tcs []model.Torrent,
	failedHashes []protocol.ID,
	errs []error,
	params MessageParams,
) error {
	if len(failedHashes) > 0 {
		if len(tcs) == 0 {
			return errors.Join(errs...)
		}

		key := ""
		if len(failedHashes) > 0 {
			key = failedHashes[0].String()
		}

		c.kafkaProducer.Produce(kafka.TopicProcessTorrent, key, MessageParams{
			InfoHashes:   failedHashes,
			ClassifyMode: params.ClassifyMode,
			ContentType:  params.ContentType,
		})
	}

	if len(tcs) == 0 {
		return nil
	}

	if err := c.persist(ctx, persistPayload{
		torrents: tcs,
	}); err != nil {
		return err
	}

	c.publishClassified(ctx, tcs)

	enriched := make([]string, 0, len(tcs))
	for _, t := range tcs {
		enriched = append(enriched, t.InferID())
	}

	if len(enriched) > 0 {
		key := enriched[0]
		c.kafkaProducer.Produce(kafka.TopicEnriched, key, enriched)
	}

	return nil
}

// publishClassified emits webhook events for torrents that were just
// classified and persisted. It never blocks the pipeline: the publisher
// delivers asynchronously.
func (c *processor) publishClassified(ctx context.Context, tcs []model.Torrent) {
	if c.webhook == nil || len(tcs) == 0 {
		return
	}

	for _, t := range tcs {
		c.webhook.Publish(context.WithoutCancel(ctx), webhook.NewClassifiedEvent(t))
	}
}

func (c *processor) SwapRunner(runner classifier.Runner) {
	c.runnerAtomic.Store(&classifierRunnerHolder{runner: runner})
	c.logger.Infow("classifier runner swapped")
}

func (c *processor) UpdateTorrentFilter(cfg classifier.TorrentFilterConfig) error {
	fs, err := compileFilterState(cfg)
	if err != nil {
		return err
	}

	c.filter.Store(fs)
	c.logger.Infow("torrent filter updated",
		"mode", cfg.Mode,
		"title_patterns", len(cfg.TitlePatterns),
		"filename_patterns", len(cfg.FilenamePatterns),
	)

	return nil
}

func newTorrentContent(t model.Torrent, c classifier.ClassificationResult) model.Torrent {
	var filesCount model.NullUint
	if t.FilesCount.Valid {
		filesCount = t.FilesCount
	}

	t.ContentType = c.ContentType
	t.Languages = c.Languages
	t.FilesCount = filesCount

	t.ContentSource = model.NullString{}
	t.ContentID = model.NullString{}
	t.Content = model.Content{}

	if c.Content != nil {
		content := *c.Content
		content.UpdateTsv()
		t.ContentType = model.NewNullContentType(content.Type)
		t.ContentSource = model.NewNullString(content.Source)
		t.ContentID = model.NewNullString(content.ID)
		t.Content = content
	}

	t.UpdateTsv()

	return t
}
