package torrentstore

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hexsans/hexmagnet/internal/configmgr"
	"github.com/hexsans/hexmagnet/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPathFor(t *testing.T) {
	t.Parallel()

	s := New(func() string { return "/data/torrents" })
	id := testutil.MustParseID("aabbccddeeff00112233445566778899aabbccdd")

	path := s.PathFor(id)
	rel, err := filepath.Rel("/data/torrents", path)
	require.NoError(t, err)

	assert.Equal(t, filepath.Join("aab", "bcc", "dde", "eff", "001", "122", "aabbccddeeff00112233445566778899aabbccdd.torrent"), rel)
}

func TestPutGetRoundtrip(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	s := New(func() string { return root })
	id := testutil.MustParseID("aabbccddeeff00112233445566778899aabbccdd")
	data := []byte("d4:infod4:name3:abce")

	err := s.Put(context.Background(), id, data)
	require.NoError(t, err)

	got, err := s.Get(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, data, got)

	// file exists at the expected sharded location
	_, statErr := os.Stat(s.PathFor(id))
	require.NoError(t, statErr)

	// no temp files left behind
	entries, err := os.ReadDir(filepath.Join(root, "aab", "bcc", "dde", "eff", "001", "122"))
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.False(t, strings.HasPrefix(entries[0].Name(), ".tmp-"))
}

func TestGetNotFound(t *testing.T) {
	t.Parallel()

	s := New(func() string { return t.TempDir() })
	id := testutil.MustParseID("aabbccddeeff00112233445566778899aabbccdd")

	_, err := s.Get(context.Background(), id)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestStoreNoRootConfigured(t *testing.T) {
	t.Parallel()

	s := New(func() string { return "" })
	id := testutil.MustParseID("aabbccddeeff00112233445566778899aabbccdd")

	_, err := s.Get(context.Background(), id)
	assert.Error(t, err)
}

func TestEnsureTorrentDir(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	dir := filepath.Join(base, "data", "torrents")

	s := New(func() string { return dir })

	require.NoError(t, s.EnsureTorrentDir())

	fi, err := os.Stat(dir)
	require.NoError(t, err)
	assert.True(t, fi.IsDir())

	require.NoError(t, s.EnsureTorrentDir())
}

func TestEnsureTorrentDir_NoDirConfigured(t *testing.T) {
	t.Parallel()

	s := New(func() string { return "" })
	assert.Error(t, s.EnsureTorrentDir())
}

func TestEnsureTorrentDir_FollowsConfigUpdate(t *testing.T) {
	t.Parallel()

	dir1 := filepath.Join(t.TempDir(), "a")
	dir2 := filepath.Join(t.TempDir(), "b")

	s := New(func() string { return dir1 })
	cm := configmgr.NewManager(&configmgr.Snapshot{}, nil)
	cm.Subscribe("torrent_store",
		func(context.Context, *configmgr.Snapshot) error { return s.EnsureTorrentDir() },
		configmgr.ApplyAsync)

	s.dir = func() string { return dir2 }

	require.NoError(t, cm.Apply(context.Background(), &configmgr.Snapshot{}))
	assert.Eventually(t, func() bool {
		fi, err := os.Stat(dir2)
		return err == nil && fi.IsDir()
	}, 2*time.Second, 10*time.Millisecond)
}
