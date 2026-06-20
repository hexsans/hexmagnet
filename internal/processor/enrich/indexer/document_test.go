package indexer

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/hexsans/hexmagnet/internal/model"
	"github.com/hexsans/hexmagnet/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDocument_Basic(t *testing.T) {
	t.Parallel()

	infoHash := testutil.MustParseID("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	now := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)

	torrent := model.Torrent{
		InfoHash:  infoHash,
		Name:      "Test Torrent",
		Size:      1000,
		Private:   false,
		CreatedAt: now,
		UpdatedAt: now,
	}

	doc := NewDocument(torrent)
	assert.Equal(t, infoHash.String(), doc.ID)
	assert.Equal(t, infoHash.String(), doc.InfoHash)
	assert.Equal(t, "Test Torrent", doc.Name)
	assert.Equal(t, uint64(1000), doc.Size)
	assert.False(t, doc.Private)
	assert.Equal(t, "2024-06-15T10:30:00Z", doc.CreatedAt)
	assert.Equal(t, "2024-06-15T10:30:00Z", doc.UpdatedAt)
}

func TestNewDocument_WithSeedersLeechers(t *testing.T) {
	t.Parallel()

	seeders := uint(10)
	leechers := uint(5)
	torrent := model.Torrent{
		InfoHash: testutil.MustParseID("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		Seeders:  model.NullUint{Valid: true, Uint: seeders},
		Leechers: model.NullUint{Valid: true, Uint: leechers},
	}

	doc := NewDocument(torrent)
	require.NotNil(t, doc.Seeders)
	assert.Equal(t, seeders, *doc.Seeders)
	require.NotNil(t, doc.Leechers)
	assert.Equal(t, leechers, *doc.Leechers)
}

func TestNewDocument_WithFilesCount(t *testing.T) {
	t.Parallel()

	filesCount := uint(42)
	torrent := model.Torrent{
		InfoHash:   testutil.MustParseID("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		FilesCount: model.NullUint{Valid: true, Uint: filesCount},
	}

	doc := NewDocument(torrent)
	require.NotNil(t, doc.FilesCount)
	assert.Equal(t, filesCount, *doc.FilesCount)
}

func TestNewDocument_WithContentType(t *testing.T) {
	t.Parallel()

	torrent := model.Torrent{
		InfoHash:    testutil.MustParseID("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		ContentType: model.NewNullContentType(model.ContentTypeMovie),
	}

	doc := NewDocument(torrent)
	assert.Equal(t, "movie", doc.ContentType)
}

func TestNewDocument_UnknownContentType(t *testing.T) {
	t.Parallel()

	torrent := model.Torrent{
		InfoHash: testutil.MustParseID("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
	}

	doc := NewDocument(torrent)
	assert.Equal(t, "unknown", doc.ContentType)
}

func TestNewDocument_WithContentSourceID(t *testing.T) {
	t.Parallel()

	torrent := model.Torrent{
		InfoHash:      testutil.MustParseID("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		ContentSource: model.NewNullString("tmdb"),
		ContentID:     model.NewNullString("123"),
	}

	doc := NewDocument(torrent)
	assert.Equal(t, "tmdb", doc.ContentSource)
	assert.Equal(t, "123", doc.ContentID)
}

func TestNewDocument_WithLanguages(t *testing.T) {
	t.Parallel()

	torrent := model.Torrent{
		InfoHash:  testutil.MustParseID("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		Languages: model.Languages{"en": {}, "fr": {}},
	}

	doc := NewDocument(torrent)
	assert.ElementsMatch(t, []string{"en", "fr"}, doc.Languages)
}

func TestNewDocument_WithContent(t *testing.T) {
	t.Parallel()

	torrent := model.Torrent{
		InfoHash:      testutil.MustParseID("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		ContentSource: model.NewNullString("tmdb"),
		ContentID:     model.NewNullString("123"),
		Content: model.Content{
			Type:     model.ContentTypeMovie,
			Source:   "tmdb",
			ID:       "123",
			Title:    "Test Movie",
			Overview: model.NewNullString("A test movie"),
		},
	}

	doc := NewDocument(torrent)
	assert.Equal(t, "Test Movie", doc.Title)
	assert.Equal(t, "A test movie", doc.Overview)
}

func TestNewDocument_WithFiles(t *testing.T) {
	t.Parallel()

	torrent := model.Torrent{
		InfoHash: testutil.MustParseID("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		Files: []model.TorrentFile{
			{PathParts: []string{"video.mp4"}, Extension: model.NewNullString("mp4"), Size: 500},
			{PathParts: []string{"subs", "en.srt"}, Extension: model.NewNullString("srt"), Size: 10},
		},
	}

	doc := NewDocument(torrent)
	require.Len(t, doc.Files, 2)
	assert.Equal(t, []string{"video.mp4"}, doc.Files[0].PathParts)
	assert.Equal(t, "mp4", doc.Files[0].Extension)
	assert.Equal(t, uint64(500), doc.Files[0].Size)
}

func TestIndexMapping_ValidJSON(t *testing.T) {
	t.Parallel()

	mapping := IndexMapping(1024)
	assert.True(t, json.Valid([]byte(mapping)))
}

func TestParseCompositeID_Valid(t *testing.T) {
	t.Parallel()

	infoHash, contentType, contentSource, contentID, err := parseCompositeID("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa:movie:tmdb:123")
	require.NoError(t, err)
	assert.Equal(t, testutil.MustParseID("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"), infoHash)
	assert.Equal(t, "movie", contentType)
	assert.Equal(t, "tmdb", contentSource)
	assert.Equal(t, "123", contentID)
}

func TestParseCompositeID_TooFewParts(t *testing.T) {
	t.Parallel()

	_, _, _, _, err := parseCompositeID("aaa:bbb")
	assert.ErrorContains(t, err, "invalid composite ID")
}

func TestParseCompositeID_InvalidInfoHash(t *testing.T) {
	t.Parallel()

	_, _, _, _, err := parseCompositeID("invalid:movie:tmdb:123")
	assert.ErrorContains(t, err, "invalid info hash")
}

func TestBulkBody_SingleDoc(t *testing.T) {
	t.Parallel()

	docs := []TorrentContentDocument{
		{
			ID:       "aaa",
			InfoHash: "aaa",
			Name:     "Test",
			Size:     100,
		},
	}

	data, err := BulkBody(docs)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	require.Len(t, lines, 2)

	var meta map[string]any

	err = json.Unmarshal([]byte(lines[0]), &meta)
	require.NoError(t, err)

	indexMeta, ok := meta["index"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "torrent_content", indexMeta["_index"])
	assert.Equal(t, "aaa", indexMeta["_id"])

	var doc TorrentContentDocument

	err = json.Unmarshal([]byte(lines[1]), &doc)
	require.NoError(t, err)
	assert.Equal(t, "Test", doc.Name)
}

func TestBulkBody_MultipleDocs(t *testing.T) {
	t.Parallel()

	docs := []TorrentContentDocument{
		{ID: "aaa", InfoHash: "aaa", Name: "First"},
		{ID: "bbb", InfoHash: "bbb", Name: "Second"},
	}

	data, err := BulkBody(docs)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	require.Len(t, lines, 4)
}

func TestBulkBody_EmptyDocs(t *testing.T) {
	t.Parallel()

	data, err := BulkBody(nil)
	require.NoError(t, err)
	assert.Empty(t, data)
}

func TestBuildSearchText_WithContentMetadata(t *testing.T) {
	t.Parallel()

	year := uint16(2023)
	doc := TorrentContentDocument{
		Name:        "Fake.Movie.Title.2023.1080p.BluRay.x264-FAKE.mkv",
		Title:       "Fake Movie Title",
		Overview:    "A completely fictional movie about nothing in particular",
		ContentType: "movie",
		ReleaseYear: &year,
	}

	result := BuildSearchText(doc)
	assert.Contains(t, result, "Fake Movie Title")
	assert.Contains(t, result, "movie")
	assert.Contains(t, result, "2023")
	assert.Contains(t, result, "A completely fictional movie about nothing in particular")
	assert.Contains(t, result, "Fake.Movie.Title.2023.1080p.BluRay.x264-FAKE")
}

func TestBuildSearchText_RawNameOnly(t *testing.T) {
	t.Parallel()

	doc := TorrentContentDocument{
		Name: "Placeholder.Name.2024.1080p.WEB-DL.AAC2.0.x264-FAKEGROUP.mkv",
	}

	result := BuildSearchText(doc)
	assert.Contains(t, result, "Placeholder.Name.2024.1080p.WEB-DL.AAC2.0.x264-FAKEGROUP")
	assert.NotContains(t, result, ".mkv")
}

func TestBuildSearchText_TitleSameAsName(t *testing.T) {
	t.Parallel()

	doc := TorrentContentDocument{
		Name:  "Nonsense Title",
		Title: "Nonsense Title",
	}

	result := BuildSearchText(doc)
	assert.Contains(t, result, "Nonsense Title")
	lines := strings.Count(result, "Nonsense Title")
	assert.LessOrEqual(t, lines, 2)
}

func TestBuildSearchText_WithFiles(t *testing.T) {
	t.Parallel()

	doc := TorrentContentDocument{
		Name: "Fictional Bundle",
		Files: []File{
			{PathParts: []string{"feature.mp4"}, Size: 1000},
			{PathParts: []string{"subtitles", "en.vtt"}, Size: 10},
			{PathParts: []string{"subtitles", "de.vtt"}, Size: 10},
		},
	}

	result := BuildSearchText(doc)
	assert.Contains(t, result, "Fictional Bundle")
	assert.Contains(t, result, "feature")
	assert.Contains(t, result, "en")
}

func TestBuildSearchText_NoNameNoTitle(t *testing.T) {
	t.Parallel()

	doc := TorrentContentDocument{}

	result := BuildSearchText(doc)
	assert.Empty(t, result)
}

func TestBuildSearchText_CleanedName(t *testing.T) {
	t.Parallel()

	doc := TorrentContentDocument{
		Name: "[Foo.Bar.Baz] Totally Fake Show (2023) [1080p] [BluRay] [5.1] [FAKEGRP].mp4",
	}

	result := BuildSearchText(doc)
	assert.Contains(t, result, "Totally Fake Show")
	assert.NotContains(t, result, "[Foo.Bar.Baz]")
	assert.NotContains(t, result, "[FAKEGRP]")
}

func TestNewDocument_ReleaseYear(t *testing.T) {
	t.Parallel()

	releaseDate := model.NewDateFromTime(time.Date(2020, 5, 1, 0, 0, 0, 0, time.UTC))
	torrent := model.Torrent{
		InfoHash:      testutil.MustParseID("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		ContentSource: model.NewNullString("tmdb"),
		ContentID:     model.NewNullString("123"),
		Content: model.Content{
			Type:        model.ContentTypeMovie,
			Source:      "tmdb",
			ID:          "123",
			Title:       "Test",
			ReleaseDate: releaseDate,
		},
	}

	doc := NewDocument(torrent)
	require.NotNil(t, doc.ReleaseYear)
	assert.Equal(t, uint16(2020), *doc.ReleaseYear)
}

func TestNewDocument_NullSeedersLeechers(t *testing.T) {
	t.Parallel()

	torrent := model.Torrent{
		InfoHash: testutil.MustParseID("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
	}

	doc := NewDocument(torrent)
	assert.Nil(t, doc.Seeders)
	assert.Nil(t, doc.Leechers)
}
