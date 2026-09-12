package jobcontrol

import (
	"sort"
	"strings"
	"sync"
)

const (
	JobReindex    = "embedding reindex"
	JobReclassify = "classifier reclassify"
)

// Controller coordinates long-running maintenance jobs. Only one job may be
// active at a time; while any job is active the DHT crawler is paused. Callers
// acquire a slot before starting and release it when finished.
type Controller struct {
	mu     sync.Mutex
	active map[string]struct{}
}

func NewController() *Controller {
	return &Controller{
		active: make(map[string]struct{}),
	}
}

// TryAcquire reserves the single job slot for name. It returns false when
// another job is already running.
func (c *Controller) TryAcquire(name string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.active) > 0 {
		return false
	}

	c.active[name] = struct{}{}

	return true
}

func (c *Controller) Release(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.active, name)
}

// Paused reports whether any maintenance job is currently running.
func (c *Controller) Paused() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	return len(c.active) > 0
}

// Reason returns the active job name(s), or an empty string when idle.
func (c *Controller) Reason() string {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.active) == 0 {
		return ""
	}

	names := make([]string, 0, len(c.active))
	for name := range c.active {
		names = append(names, name)
	}

	sort.Strings(names)

	return strings.Join(names, ", ")
}
