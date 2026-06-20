package persist

import (
	"context"
	"testing"

	"github.com/hexsans/hexmagnet/internal/dht"
	"github.com/hexsans/hexmagnet/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	h := New(testutil.NewTestLogger())
	assert.NotNil(t, h)
}

func TestHandlePersist_ValidHash(t *testing.T) {
	t.Parallel()

	h := New(testutil.NewTestLogger())
	msg := dht.MetaInfoMessage{
		InfoHash: "0123456789abcdef0123456789abcdef01234567",
		Node:     "1.2.3.4:6881",
	}

	result, err := h.HandlePersist(context.Background(), msg)
	require.NoError(t, err)
	assert.True(t, result.Scrape)
	assert.Equal(t, msg.InfoHash, result.InfoHash)
}

func TestHandlePersist_InvalidHash(t *testing.T) {
	t.Parallel()

	h := New(testutil.NewTestLogger())
	msg := dht.MetaInfoMessage{
		InfoHash: "invalid",
		Node:     "1.2.3.4:6881",
	}

	_, err := h.HandlePersist(context.Background(), msg)
	assert.Error(t, err)
}

func TestHandlePersist_EmptyHash(t *testing.T) {
	t.Parallel()

	h := New(testutil.NewTestLogger())
	msg := dht.MetaInfoMessage{
		InfoHash: "",
		Node:     "1.2.3.4:6881",
	}

	_, err := h.HandlePersist(context.Background(), msg)
	assert.Error(t, err)
}

func TestHandlePersist_ShortHash(t *testing.T) {
	t.Parallel()

	h := New(testutil.NewTestLogger())
	msg := dht.MetaInfoMessage{
		InfoHash: "abcdef",
		Node:     "1.2.3.4:6881",
	}

	_, err := h.HandlePersist(context.Background(), msg)
	assert.Error(t, err)
}
