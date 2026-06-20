package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBucketDurationValues(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "minute", string(BucketDuration("minute")))
	assert.Equal(t, "hour", string(BucketDuration("hour")))
	assert.Equal(t, "day", string(BucketDuration("day")))
}
