package search

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTorrentSearchFieldValues(t *testing.T) {
	t.Parallel()

	assert.Equal(t, FieldCreatedAt, TorrentSearchField("created_at"))
	assert.Equal(t, FieldUpdatedAt, TorrentSearchField("updated_at"))
	assert.Equal(t, FieldSize, TorrentSearchField("size"))
	assert.Equal(t, FieldFilesCount, TorrentSearchField("files_count"))
	assert.Equal(t, FieldSeeders, TorrentSearchField("seeders"))
	assert.Equal(t, FieldLeechers, TorrentSearchField("leechers"))
	assert.Equal(t, FieldName, TorrentSearchField("name"))
	assert.Equal(t, FieldInfoHash, TorrentSearchField("info_hash"))
	assert.Equal(t, FieldRelevance, TorrentSearchField("relevance"))
}

func TestSortDirectionValues(t *testing.T) {
	t.Parallel()

	assert.Equal(t, SortAsc, SortDirection("asc"))
	assert.Equal(t, SortDesc, SortDirection("desc"))
}
