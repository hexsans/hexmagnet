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
	"github.com/hexsans/hexmagnet/internal/filter"
	"github.com/hexsans/hexmagnet/internal/model"
	"github.com/hexsans/hexmagnet/internal/protocol"
	"github.com/hexsans/hexmagnet/internal/queue"
	"github.com/hexsans/hexmagnet/internal/queue/kafka"
	"github.com/hexsans/hexmagnet/internal/retryqueue"
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
	patterns, err := filter.Compile(cfg.TitlePatterns, cfg.FilenamePatterns)
	if err != nil {
		return nil, err
	}

	return &filterState{
		mode:             cfg.Mode,
		titlePatterns:    patterns.Titles(),
		filenamePatterns: patterns.Filenames(),
	}, nil
}

const classifyFailureSampleSize = 3

type processor struct {
	defaultWorkflow    string
	searchRuntime      *dbsearch.Runtime
	runner             classifier.Runner
	runnerAtomic       atomic.Pointer[classifierRunnerHolder]
	queries            *db.Queries
	blockingManager    blocking.Manager
	kafkaProducer      queue.Producer
	webhook            *webhook.Publisher
	retryQueue         *retryqueue.Queue
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

func (c *processor) Process(ctx context.Context, params MessageParams) error {
	workflowName := c.defaultWorkflow

	searchResult, err := c.fetchAndJoinData(ctx, params)
	if err != nil {
		return err
	}

	var (
		mtx sync.Mutex
		wg  sync.WaitGroup
	)

	tcs := make([]model.Torrent, 0, len(searchResult.Torrents))

	missingHashes := make([]protocol.ID, 0, len(searchResult.MissingInfoHashes))
	for _, h := range searchResult.MissingInfoHashes {
		missingHashes = append(missingHashes, db.ToProtocolID(h))
	}

	classifyFailures := make([]classifyFailure, 0, len(searchResult.Torrents))

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

			if params.ClassifyMode != ClassifyModeRematch && c.wasRecentlyClassified(torrent.InfoHash) {
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

				err := fmt.Errorf("unexpected type from singleflight: %T", v)
				classifyFailures = append(classifyFailures, classifyFailure{
					infoHash: torrent.InfoHash,
					err:      err,
				})
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

				classifyFailures = append(classifyFailures, classifyFailure{
					infoHash: torrent.InfoHash,
					err:      classifyErr,
				})
			} else {
				torrentContent := newTorrentContent(torrent, cl, c.maxSearchFiles())

				if params.ContentType != "" {
					torrentContent.ContentType = model.NewNullContentType(params.ContentType)
				}

				tcs = append(tcs, torrentContent)
			}
			mtx.Unlock()
		}(torrent)
	}

	wg.Wait()

	return c.handleClassified(ctx, tcs, missingHashes, classifyFailures, params)
}

// classifyFailure pairs a failed info hash with the error that caused it.
type classifyFailure struct {
	infoHash protocol.ID
	err      error
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

	match := filter.AnyMatch(fs.titlePatterns, torrent.Name)

	if len(fs.filenamePatterns) > 0 && !match {
		for _, f := range torrent.Files {
			if filter.AnyMatch(fs.filenamePatterns, strings.Join(f.PathParts, "/")) {
				match = true

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
	missingHashes []protocol.ID,
	classifyFailures []classifyFailure,
	params MessageParams,
) error {
	c.logClassifySummary(tcs, missingHashes, classifyFailures)

	// Torrent rows that no longer exist cannot be classified. Re-producing
	// messages for them would loop forever, so they are dropped here.
	if len(missingHashes) > 0 {
		c.logger.Debugw("dropping missing torrents from processed message",
			"count", len(missingHashes),
			"info_hash", missingHashes[0].String(),
		)
	}

	if len(classifyFailures) > 0 {
		retryable, pauseErr := splitClassifyFailures(classifyFailures)

		if len(retryable) > 0 {
			c.enqueueClassifyRetries(ctx, retryable, params)
		}

		if pauseErr != nil {
			return pauseErr
		}
	}

	if len(tcs) == 0 {
		return nil
	}

	if err := c.persist(ctx, persistPayload{
		torrents: tcs,
	}); err != nil {
		return err
	}

	if c.retryQueue != nil {
		classified := make([]string, 0, len(tcs))
		for _, t := range tcs {
			classified = append(classified, t.InfoHash.String())
		}

		c.retryQueue.Remove(ctx, classified...)
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

// logClassifySummary emits one debug line per processed message instead of
// one line per torrent, keeping the batch outcome visible without flooding
// the log when many torrents fail at once.
func (c *processor) logClassifySummary(
	tcs []model.Torrent,
	missingHashes []protocol.ID,
	classifyFailures []classifyFailure,
) {
	if len(missingHashes) == 0 && len(classifyFailures) == 0 {
		return
	}

	samples := make([]string, 0, classifyFailureSampleSize)
	for _, f := range classifyFailures {
		if len(samples) >= classifyFailureSampleSize {
			break
		}

		samples = append(samples, f.infoHash.String()+": "+f.err.Error())
	}

	c.logger.Debugw("classification batch completed with failures",
		"classified", len(tcs),
		"failed", len(classifyFailures),
		"missing", len(missingHashes),
		"samples", samples,
	)
}

// enqueueClassifyRetries records classification failures in the retry queue
// as a single batched upsert, keyed per info hash with the error that caused
// the failure.
func (c *processor) enqueueClassifyRetries(
	ctx context.Context,
	failures []classifyFailure,
	params MessageParams,
) {
	if c.retryQueue == nil {
		return
	}

	items := make([]retryqueue.RetryItem, 0, len(failures))
	for _, f := range failures {
		items = append(items, retryqueue.RetryItem{
			Stage:    retryqueue.StageClassify,
			InfoHash: f.infoHash.String(),
			Payload: MessageParams{
				InfoHashes:   []protocol.ID{f.infoHash},
				ClassifyMode: params.ClassifyMode,
				ContentType:  params.ContentType,
			},
		})
	}

	if err := c.retryQueue.EnqueueBatch(ctx, items, errors.Join(classifyFailureErrs(failures)...)); err != nil {
		c.logger.Errorw("failed to enqueue classify retry entries",
			"count", len(failures),
			"error", err,
		)
	}
}

func classifyFailureErrs(failures []classifyFailure) []error {
	errs := make([]error, 0, len(failures))
	for _, f := range failures {
		errs = append(errs, f.err)
	}

	return errs
}

// splitClassifyFailures separates failures that should be retried through the
// retry queue from LLM outages. The latter must pause the calling batch job
// (strict LLM mode) instead of being silently retried or downgraded to rules.
func splitClassifyFailures(failures []classifyFailure) ([]classifyFailure, error) {
	var (
		retryable []classifyFailure
		pauseErr  error
	)

	for _, f := range failures {
		var llmErr *classifier.LLMClassifyError
		if errors.As(f.err, &llmErr) {
			if pauseErr == nil {
				pauseErr = llmErr
			}

			continue
		}

		retryable = append(retryable, f)
	}

	return retryable, pauseErr
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

// maxSearchFiles returns the configured cap on file paths included in the
// torrent search vector (<= 0 means no cap).
func (c *processor) maxSearchFiles() int {
	if c.searchRuntime == nil || c.searchRuntime.SearchConfig == nil {
		return 0
	}

	return c.searchRuntime.SearchConfig.Get().MaxSearchFiles
}

func newTorrentContent(t model.Torrent, c classifier.ClassificationResult, maxSearchFiles int) model.Torrent {
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

	t.UpdateTsv(maxSearchFiles)

	return t
}
