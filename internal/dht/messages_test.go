package dht

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDiscoveredHash(t *testing.T) {
	t.Parallel()

	now := time.Now()
	msg := DiscoveredHash{
		InfoHash:     "0123456789abcdef0123456789abcdef01234567",
		Node:         "1.2.3.4:6881",
		DiscoveredAt: now,
	}
	assert.Equal(t, "0123456789abcdef0123456789abcdef01234567", msg.InfoHash)
	assert.Equal(t, "1.2.3.4:6881", msg.Node)
	assert.Equal(t, now, msg.DiscoveredAt)
}

func TestGetPeersMessage(t *testing.T) {
	t.Parallel()

	msg := GetPeersMessage{
		InfoHash: "infohash1234567890abcdef01234567",
		Node:     "5.6.7.8:6881",
	}
	assert.Equal(t, "infohash1234567890abcdef01234567", msg.InfoHash)
	assert.Equal(t, "5.6.7.8:6881", msg.Node)
}

func TestScrapeMessage(t *testing.T) {
	t.Parallel()

	msg := ScrapeMessage{
		InfoHash: "scrapehash1234567890abcdef0123456",
		Node:     "9.10.11.12:6881",
	}
	assert.Equal(t, "scrapehash1234567890abcdef0123456", msg.InfoHash)
	assert.Equal(t, "9.10.11.12:6881", msg.Node)
}

func TestMetainfoFile(t *testing.T) {
	t.Parallel()

	f := MetainfoFile{
		PathParts: []string{"dir", "file.txt"},
		Size:      1024,
	}
	assert.Equal(t, []string{"dir", "file.txt"}, f.PathParts)
	assert.Equal(t, uint64(1024), f.Size)
}

func TestMetaInfoMessage(t *testing.T) {
	t.Parallel()

	msg := MetaInfoMessage{
		InfoHash:     "metainfohash1234567890abcdef01234",
		RawInfoBytes: []byte{0x01, 0x02, 0x03},
		Name:         "test-torrent",
		Private:      true,
		Files:        []MetainfoFile{{PathParts: []string{"f"}, Size: 512}},
		TotalSize:    512,
		Node:         "1.2.3.4:6881",
	}
	assert.Equal(t, "metainfohash1234567890abcdef01234", msg.InfoHash)
	assert.Equal(t, []byte{0x01, 0x02, 0x03}, msg.RawInfoBytes)
	assert.Equal(t, "test-torrent", msg.Name)
	assert.True(t, msg.Private)
	assert.Len(t, msg.Files, 1)
	assert.Equal(t, uint64(512), msg.TotalSize)
	assert.Equal(t, "1.2.3.4:6881", msg.Node)
}

func TestScrapeResultMessage(t *testing.T) {
	t.Parallel()

	now := time.Now()
	msg := ScrapeResultMessage{
		InfoHash:  "scraperesulthash1234567890abcdef0",
		Seeders:   10,
		Leechers:  5,
		ScrapedAt: now,
	}
	assert.Equal(t, "scraperesulthash1234567890abcdef0", msg.InfoHash)
	assert.Equal(t, uint(10), msg.Seeders)
	assert.Equal(t, uint(5), msg.Leechers)
	assert.Equal(t, now, msg.ScrapedAt)
}
