package version

import "sync"

// GitTag is set at build time via -ldflags -X.
var GitTag string

var gitTagMu sync.RWMutex

// GitTagValue returns the current GitTag value.
func GitTagValue() string {
	gitTagMu.RLock()
	defer gitTagMu.RUnlock()

	return GitTag
}
