// Package webhook publishes classified torrent events to external HTTP
// endpoints. It lets consumers (custom scripts, download clients, automation
// engines) react to newly crawled content without polling the search API.
package webhook

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/hexsans/hexmagnet/internal/model"
)

// Event types supported by the publisher.
const (
	// EventClassified is emitted after a torrent has been classified and
	// persisted, and carries the classification result (content type, ids...).
	EventClassified = "classified"
)

var supportedEvents = []string{EventClassified}

const (
	defaultTimeout    = 10 * time.Second
	defaultMaxRetries = 3
	defaultQueueSize  = 1000
)

// Config controls the webhook publisher.
type Config struct {
	// Enabled turns publishing on or off.
	Enabled bool `yaml:"enabled"`
	// Urls is the list of endpoints that receive every event.
	Urls []string `yaml:"urls"`
	// Events filters which event types are sent. Empty = all supported events.
	Events []string `yaml:"events"`
	// Categories, when non-empty, restricts delivery to torrents whose
	// classified content type (movie, tv_show, music, ...) is in the list.
	// Empty = no category filter.
	Categories []string `yaml:"categories"`
	// TitlePatterns, when non-empty, restricts delivery to torrents whose
	// name matches at least one of the (case-insensitive) RE2 patterns.
	// Empty = no title filter.
	TitlePatterns []string `yaml:"title_patterns"`
	// FilenamePatterns, when non-empty, restricts delivery to torrents that
	// have at least one file whose path (parts joined with "/") matches at
	// least one of the (case-insensitive) RE2 patterns. Empty = no filename
	// filter.
	FilenamePatterns []string `yaml:"filename_patterns"`
	// Timeout is the per-request timeout.
	Timeout time.Duration `yaml:"timeout"`
	// MaxRetries is how many times a failed delivery is retried before it is
	// dropped (retries use exponential backoff). The first attempt is not a
	// retry, so 3 means up to 3 attempts total.
	MaxRetries int `yaml:"max_retries"`
	// BaseURL, when set, is the public base URL of the HexMagnet HTTP server.
	// It is used to build the torrent download link in the payload. When empty
	// the link is omitted.
	BaseURL string `yaml:"base_url"`
	// Headers are extra HTTP headers sent with every delivery, e.g.
	// Authorization: Bearer <token> for endpoints that require auth.
	Headers map[string]string `yaml:"headers"`
	// QueueSize is the capacity of the async delivery queue. When full, new
	// events are dropped (and logged) rather than blocking the pipeline.
	QueueSize int `yaml:"queue_size"`
}

func NewDefaultConfig() Config {
	return Config{
		Enabled:          false,
		Urls:             []string{},
		Events:           []string{EventClassified},
		Categories:       []string{},
		TitlePatterns:    []string{},
		FilenamePatterns: []string{},
		Timeout:          defaultTimeout,
		MaxRetries:       defaultMaxRetries,
		BaseURL:          "",
		Headers:          map[string]string{},
		QueueSize:        defaultQueueSize,
	}
}

// Validate checks the config values and returns a descriptive error listing
// every problem found.
func (c Config) Validate() error {
	var errs []error

	if c.Timeout <= 0 {
		errs = append(errs, fmt.Errorf("timeout must be greater than zero, got %s", c.Timeout))
	}

	if c.MaxRetries < 0 || c.MaxRetries > 10 {
		errs = append(errs, fmt.Errorf("max_retries must be between 0 and 10, got %d", c.MaxRetries))
	}

	if c.QueueSize < 1 {
		errs = append(errs, fmt.Errorf("queue_size must be at least 1, got %d", c.QueueSize))
	}

	for _, u := range c.Urls {
		parsed, err := url.Parse(u)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			errs = append(errs, fmt.Errorf("invalid webhook url %q (expected an absolute http(s) URL)", u))
		}
	}

	for _, e := range c.Events {
		if !strings.EqualFold(e, EventClassified) {
			errs = append(errs, fmt.Errorf("unsupported webhook event %q (supported: %s)", e, strings.Join(supportedEvents, ", ")))
		}
	}

	for _, cat := range c.Categories {
		if _, err := model.ParseContentType(cat); err != nil {
			supported := strings.Join(model.ContentTypeNames(), ", ")
			errs = append(errs, fmt.Errorf("unsupported webhook category %q (supported: %s)", cat, supported))
		}
	}

	for _, p := range c.TitlePatterns {
		if _, err := regexp.Compile("(?i)" + p); err != nil {
			errs = append(errs, fmt.Errorf("invalid title pattern %q: %w", p, err))
		}
	}

	for _, p := range c.FilenamePatterns {
		if _, err := regexp.Compile("(?i)" + p); err != nil {
			errs = append(errs, fmt.Errorf("invalid filename pattern %q: %w", p, err))
		}
	}

	if c.BaseURL != "" {
		parsed, err := url.Parse(c.BaseURL)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			errs = append(errs, fmt.Errorf("invalid base_url %q (expected an absolute http(s) URL)", c.BaseURL))
		}
	}

	for k, v := range c.Headers {
		if strings.TrimSpace(k) == "" || strings.TrimSpace(v) == "" {
			errs = append(errs, fmt.Errorf("invalid header %q (key and value must be non-empty)", k))
		}
	}

	return joinErrors(errs)
}

func joinErrors(errs []error) error {
	if len(errs) == 0 {
		return nil
	}

	msgs := make([]string, len(errs))
	for i, e := range errs {
		msgs[i] = e.Error()
	}

	return fmt.Errorf("webhooks: %s", strings.Join(msgs, "; "))
}
