package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGitTagDefault(t *testing.T) {
	t.Parallel()

	assert.Empty(t, GitTagValue())
}

func TestGitTagSet(t *testing.T) {
	t.Parallel()

	gitTagMu.Lock()
	original := GitTag
	GitTag = "v1.0.0"

	defer func() {
		GitTag = original
		gitTagMu.Unlock()
	}()

	assert.Equal(t, "v1.0.0", GitTag)
}

func TestUserAgent(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "HexMagnet", formatUserAgent(""))
	assert.Equal(t, "HexMagnet/v1.2.3", formatUserAgent("v1.2.3"))
}

func TestNewHealthCheck(t *testing.T) {
	t.Parallel()

	result := NewHealthCheck()
	assert.NotNil(t, result.HealthOption)
}

func TestNewModule(t *testing.T) {
	t.Parallel()

	opt := NewModule()
	assert.NotNil(t, opt)
}
