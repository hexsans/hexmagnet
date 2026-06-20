package torrentstore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hexsans/hexmagnet/internal/protocol"
)

var ErrNotFound = errors.New("torrent file not found")

type Store struct {
	dir func() string
}

func New(dir func() string) *Store {
	return &Store{dir: dir}
}

// PathFor returns the on-disk path for the given info hash:
// <dir>/<aaa>/<bbb>/<ccc>/<ddd>/<eee>/<fff>/<infohash>.torrent
func (s *Store) PathFor(id protocol.ID) string {
	h := id.String()
	return filepath.Join(s.dir(), h[0:3], h[3:6], h[6:9], h[9:12], h[12:15], h[15:18], h+".torrent")
}

// resolveDir returns the currently configured storage directory. It is
// re-read on every call so config changes take effect immediately.
func (s *Store) resolveDir() (string, error) {
	dir := s.dir()
	if dir == "" {
		return "", errors.New("torrent file path not configured")
	}

	return dir, nil
}

// EnsureTorrentDir creates the configured storage directory if it does not
// already exist. No-op when it exists.
func (s *Store) EnsureTorrentDir() error {
	dir, err := s.resolveDir()
	if err != nil {
		return err
	}

	return os.MkdirAll(dir, 0o755)
}

// Put writes the torrent file atomically (temp file + rename).
func (s *Store) Put(_ context.Context, id protocol.ID, data []byte) error {
	path := s.PathFor(id)
	if _, err := s.resolveDir(); err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create torrent dir %s: %w", dir, err)
	}

	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file in %s: %w", dir, err)
	}

	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp file: %w", err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("rename temp file to %s: %w", path, err)
	}

	return nil
}

// Get reads the torrent file for the given info hash.
func (s *Store) Get(_ context.Context, id protocol.ID) ([]byte, error) {
	if _, err := s.resolveDir(); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(s.PathFor(id))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}

		return nil, err
	}

	return data, nil
}

// Has reports whether a torrent file exists for the given info hash.
func (s *Store) Has(id protocol.ID) bool {
	if _, err := s.resolveDir(); err != nil {
		return false
	}

	fi, err := os.Stat(s.PathFor(id))

	return err == nil && !fi.IsDir()
}
