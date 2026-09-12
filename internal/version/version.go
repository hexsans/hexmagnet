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

const userAgentName = "HexMagnet"

// UserAgent returns the User-Agent header value identifying this service and
// its build version.
func UserAgent() string {
	return formatUserAgent(GitTagValue())
}

func formatUserAgent(gitTag string) string {
	if gitTag != "" {
		return userAgentName + "/" + gitTag
	}

	return userAgentName
}
