package webhook

import (
	"bytes"
	"context"
	"fmt"
	"maps"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hexsans/hexmagnet/internal/version"
	"github.com/hexsans/hexmagnet/internal/worker"
	"go.uber.org/zap"
)

// compiledFilter holds the precompiled filtering rules derived from a Config.
// Filters are applied in Publish, before an event is enqueued.
type compiledFilter struct {
	categories       map[string]struct{}
	titlePatterns    []*regexp.Regexp
	filenamePatterns []*regexp.Regexp
}

// compileFilter compiles the regex patterns of a Config. Invalid regexes are
// rejected by Config.Validate, so a failure here indicates a config applied
// behind the validator's back; callers keep the previous filter in that case.
func compileFilter(cfg Config) (*compiledFilter, error) {
	f := &compiledFilter{}

	if len(cfg.Categories) > 0 {
		f.categories = make(map[string]struct{}, len(cfg.Categories))
		for _, c := range cfg.Categories {
			f.categories[c] = struct{}{}
		}
	}

	for _, p := range cfg.TitlePatterns {
		re, err := regexp.Compile("(?i)" + p)
		if err != nil {
			return nil, fmt.Errorf("invalid title pattern %q: %w", p, err)
		}

		f.titlePatterns = append(f.titlePatterns, re)
	}

	for _, p := range cfg.FilenamePatterns {
		re, err := regexp.Compile("(?i)" + p)
		if err != nil {
			return nil, fmt.Errorf("invalid filename pattern %q: %w", p, err)
		}

		f.filenamePatterns = append(f.filenamePatterns, re)
	}

	return f, nil
}

// matches reports whether the event passes every active filter. Rules:
//   - categories, when non-empty, require the event's content type (a missing
//     content type counts as "unknown") to be in the set;
//   - title patterns, when non-empty, require at least one match against the
//     torrent name;
//   - filename patterns, when non-empty, require at least one match against
//     any file path;
//   - patterns within a list are OR-ed, the three filters are AND-ed; empty
//     filters never exclude anything.
func (f *compiledFilter) matches(e Event) bool {
	if len(f.categories) > 0 {
		ct := "unknown"
		if e.ContentType != nil {
			ct = *e.ContentType
		}

		if _, ok := f.categories[ct]; !ok {
			return false
		}
	}

	if len(f.titlePatterns) > 0 && !anyMatch(f.titlePatterns, e.Name) {
		return false
	}

	if len(f.filenamePatterns) > 0 {
		matched := false

		for _, path := range e.Files {
			if anyMatch(f.filenamePatterns, path) {
				matched = true
				break
			}
		}

		if !matched {
			return false
		}
	}

	return true
}

func anyMatch(patterns []*regexp.Regexp, s string) bool {
	for _, p := range patterns {
		if p.MatchString(s) {
			return true
		}
	}

	return false
}

// Publisher delivers events to the configured webhook Urls asynchronously.
// Publish never blocks: events are enqueued and delivered by background
// workers, so the classification pipeline is not slowed down by slow or
// unreachable endpoints.
type Publisher struct {
	config  atomic.Pointer[Config]
	filters atomic.Pointer[compiledFilter]
	logger  *zap.SugaredLogger

	// queueMu guards the queue reference. Update may swap the queue channel
	// at runtime (queue_size changes); Publish holds the read lock across the
	// enqueue so an event can never land in a channel that has already been
	// drained by a resize.
	queueMu   sync.RWMutex
	queue     chan envelope
	wake      chan struct{}
	queueSize int

	client *http.Client

	started atomic.Bool
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// envelope captures the event together with the delivery settings snapshot
// taken at Publish time, so a later config change cannot re-target or discard
// events that were already accepted.
type envelope struct {
	event   Event
	urls    []string
	timeout time.Duration
	retries int
	headers map[string]string
}

// NewPublisher creates a Publisher with the given config. The config can be
// replaced at runtime via Update.
func NewPublisher(cfg Config, logger *zap.SugaredLogger) *Publisher {
	p := &Publisher{
		queue:     make(chan envelope, queueCapacity(cfg)),
		wake:      make(chan struct{}, 1),
		queueSize: queueCapacity(cfg),
		logger:    logger,
		ctx:       context.Background(),
	}

	p.config.Store(&cfg)
	// Start from an empty (pass-through) filter so a failed initial compile
	// can never leave a nil filter behind.
	p.filters.Store(&compiledFilter{})
	p.storeFilter(cfg)

	return p
}

// Update replaces the runtime config. Urls, events, timeout, retries,
// filters and headers take effect immediately; QueueSize is applied to the
// live queue without losing pending events.
func (p *Publisher) Update(cfg Config) {
	p.config.Store(&cfg)
	p.storeFilter(cfg)
	p.resizeQueue(cfg.QueueSize)
}

// storeFilter compiles the filtering rules for cfg and stores them. When the
// regexes fail to compile (config bypassed Validate), the previous filter is
// kept and the failure is logged.
func (p *Publisher) storeFilter(cfg Config) {
	f, err := compileFilter(cfg)
	if err != nil {
		p.logger.Errorw("failed to compile webhook filters, keeping previous filter", "error", err)
		return
	}

	p.filters.Store(f)
}

// resizeQueue swaps the delivery queue for one with the given capacity,
// migrating pending events. The consumer is woken up so it re-reads the
// current queue reference.
func (p *Publisher) resizeQueue(size int) {
	size = queueCapacity(Config{QueueSize: size})

	p.queueMu.Lock()
	defer p.queueMu.Unlock()

	if cap(p.queue) == size {
		return
	}

	old := p.queue
	p.queue = make(chan envelope, size)

drain:
	for {
		select {
		case e := <-old:
			select {
			case p.queue <- e:
			default:
				p.logger.Warnw("webhook queue resize dropped event", "event", e.event.Event, "info_hash", e.event.InfoHash)
			}
		default:
			break drain
		}
	}

	p.queueSize = size

	select {
	case p.wake <- struct{}{}:
	default:
	}
}

// Publish enqueues an event for delivery. It returns immediately; delivery
// happens in the background. Events that fail the configured filters, or are
// dropped because publishing is disabled or the queue is full, are silently
// discarded (full queue is logged).
func (p *Publisher) Publish(_ context.Context, e Event) {
	cfg := p.config.Load()
	if cfg == nil || !cfg.Enabled || !cfg.eventEnabled(e.Event) || len(cfg.Urls) == 0 {
		return
	}

	if f := p.filters.Load(); f != nil && !f.matches(e) {
		return
	}

	if e.TorrentURL == "" && cfg.BaseURL != "" {
		e.TorrentURL = strings.TrimRight(cfg.BaseURL, "/") + "/api/torrents/" + e.InfoHash + "/download"
	}

	env := envelope{
		event:   e,
		urls:    slices.Clone(cfg.Urls),
		timeout: cfg.Timeout,
		retries: cfg.MaxRetries,
		headers: maps.Clone(cfg.Headers),
	}

	p.queueMu.RLock()

	select {
	case p.queue <- env:
		p.queueMu.RUnlock()
	default:
		p.queueMu.RUnlock()
		p.logger.Warnw("webhook queue full, dropping event", "event", e.Event, "info_hash", e.InfoHash)
	}
}

// eventEnabled reports whether the event type passes the configured filter.
func (c Config) eventEnabled(event string) bool {
	for _, e := range c.Events {
		if strings.EqualFold(e, event) {
			return true
		}
	}

	return false
}

// Start spawns the delivery worker. It may be called again after Stop.
func (p *Publisher) Start(ctx context.Context) error {
	if !p.started.CompareAndSwap(false, true) {
		return nil
	}

	p.ctx, p.cancel = context.WithCancel(ctx)

	p.wg.Add(1)

	worker.GoRecover(p.logger, "webhook_delivery", func() {
		defer p.wg.Done()

		p.run()
	})

	return nil
}

// run is the delivery loop. Each iteration re-reads the current queue
// reference so queue swaps are picked up.
func (p *Publisher) run() {
	for {
		select {
		case <-p.ctx.Done():
			p.drain()
			return
		default:
		}

		q := p.currentQueue()

		select {
		case e := <-q:
			p.deliver(e)
		case <-p.wake:
			// Queue was swapped; loop to pick up the new channel.
		case <-p.ctx.Done():
			p.drain()
			return
		}
	}
}

// currentQueue returns the active delivery queue.
func (p *Publisher) currentQueue() chan envelope {
	p.queueMu.RLock()
	defer p.queueMu.RUnlock()

	return p.queue
}

// drain delivers whatever is left in the queue before exiting.
func (p *Publisher) drain() {
	q := p.currentQueue()

	for {
		select {
		case e := <-q:
			p.deliver(e)
		default:
			return
		}
	}
}

// Stop cancels the delivery worker and waits for it to finish. Delivery
// attempts abort promptly once Stop is called; the wait is bounded by ctx.
func (p *Publisher) Stop(ctx context.Context) error {
	if !p.started.CompareAndSwap(true, false) {
		return nil
	}

	p.cancel()

	done := make(chan struct{})

	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// deliver sends an event to every configured URL, retrying failures with
// exponential backoff.
func (p *Publisher) deliver(e envelope) {
	body, err := e.event.JSON()
	if err != nil {
		p.logger.Errorw("failed to serialize webhook event", "error", err, "info_hash", e.event.InfoHash)
		return
	}

	attempts := e.retries + 1
	if attempts < 1 {
		attempts = 1
	}

	for _, rawURL := range e.urls {
		if err := p.deliverTo(rawURL, body, attempts, e.timeout, e.headers); err != nil {
			p.logger.Warnw("webhook delivery failed",
				"url", rawURL,
				"info_hash", e.event.InfoHash,
				"error", err,
			)
		}
	}
}

func (p *Publisher) deliverTo(rawURL string, body []byte, attempts int, timeout time.Duration, headers map[string]string) error {
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	var lastErr error

	for attempt := 1; attempt <= attempts; attempt++ {
		lastErr = p.post(p.ctx, rawURL, body, timeout, headers)
		if lastErr == nil {
			return nil
		}

		if attempt < attempts {
			backoff := time.Duration(1<<uint(attempt-1)) * 200 * time.Millisecond
			if backoff > 5*time.Second {
				backoff = 5 * time.Second
			}

			select {
			case <-time.After(backoff):
			case <-p.ctx.Done():
				return lastErr
			}
		}
	}

	return lastErr
}

func (p *Publisher) post(ctx context.Context, rawURL string, body []byte, timeout time.Duration, headers map[string]string) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rawURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", version.UserAgent())

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := p.httpClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	return fmt.Errorf("unexpected status %d", resp.StatusCode)
}

func (p *Publisher) httpClient() *http.Client {
	if p.client == nil {
		p.client = &http.Client{}
	}

	return p.client
}

func queueCapacity(cfg Config) int {
	if cfg.QueueSize < 1 {
		return defaultQueueSize
	}

	return cfg.QueueSize
}
