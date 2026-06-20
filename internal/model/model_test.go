package model

import (
	"testing"
	"time"

	"github.com/hexsans/hexmagnet/internal/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContentTypeValues(t *testing.T) {
	t.Parallel()

	vals := ContentTypeValues()
	assert.NotEmpty(t, vals)
	assert.Contains(t, vals, ContentTypeMovie)
	assert.Contains(t, vals, ContentTypeTvShow)
	assert.Contains(t, vals, ContentTypeMusic)
	assert.Contains(t, vals, ContentTypeEbook)
	assert.Contains(t, vals, ContentTypeComic)
	assert.Contains(t, vals, ContentTypeAudiobook)
	assert.Contains(t, vals, ContentTypeGame)
	assert.Contains(t, vals, ContentTypeSoftware)
	assert.Contains(t, vals, ContentTypeOther)
	assert.Contains(t, vals, ContentTypeUnknown)
	assert.Contains(t, vals, ContentTypeAdult)
	assert.Len(t, vals, 11)
}

func TestContentTypeNames(t *testing.T) {
	t.Parallel()

	names := ContentTypeNames()
	assert.NotEmpty(t, names)
	assert.Contains(t, names, "movie")
	assert.Contains(t, names, "unknown")
	assert.Len(t, names, 11)
}

func TestFacetLogicString(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "and", FacetLogicAnd.String())
	assert.Equal(t, "or", FacetLogicOr.String())
	assert.Empty(t, FacetLogic("").String())
}

func TestFacetLogicIsValid(t *testing.T) {
	t.Parallel()

	assert.True(t, FacetLogicAnd.IsValid())
	assert.True(t, FacetLogicOr.IsValid())
	assert.False(t, FacetLogic("invalid").IsValid())
	assert.False(t, FacetLogic("").IsValid())
}

func TestFacetLogicNames(t *testing.T) {
	t.Parallel()

	names := FacetLogicNames()
	assert.NotEmpty(t, names)
	assert.Contains(t, names, "and")
	assert.Contains(t, names, "or")
	assert.Len(t, names, 2)
}

func TestParseFacetLogic(t *testing.T) {
	t.Parallel()

	fl, err := ParseFacetLogic("and")
	require.NoError(t, err)
	assert.Equal(t, FacetLogicAnd, fl)

	fl, err = ParseFacetLogic("or")
	require.NoError(t, err)
	assert.Equal(t, FacetLogicOr, fl)

	fl, err = ParseFacetLogic("OR")
	require.NoError(t, err)
	assert.Equal(t, FacetLogicOr, fl)

	_, err = ParseFacetLogic("invalid")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidFacetLogic)
}

func TestFileTypeString(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "video", FileTypeVideo.String())
	assert.Equal(t, "audio", FileTypeAudio.String())
	assert.Equal(t, "archive", FileTypeArchive.String())
}

func TestFileTypeValues(t *testing.T) {
	t.Parallel()

	vals := FileTypeValues()
	assert.NotEmpty(t, vals)
	assert.Contains(t, vals, FileTypeVideo)
	assert.Contains(t, vals, FileTypeAudio)
	assert.Contains(t, vals, FileTypeArchive)
	assert.Contains(t, vals, FileTypeData)
	assert.Contains(t, vals, FileTypeDocument)
	assert.Contains(t, vals, FileTypeImage)
	assert.Contains(t, vals, FileTypeSoftware)
	assert.Contains(t, vals, FileTypeSubtitles)
	assert.Len(t, vals, 8)
}

func TestFileTypeNames(t *testing.T) {
	t.Parallel()

	names := FileTypeNames()
	assert.NotEmpty(t, names)
	assert.Contains(t, names, "video")
	assert.Contains(t, names, "audio")
	assert.Len(t, names, 8)
}

func TestLanguageString(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "en", Language("en").String())
	assert.Equal(t, "fr", Language("fr").String())
}

func TestLanguageID(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "en", Language("en").ID())
	assert.Equal(t, "fr", Language("fr").ID())
}

func TestMaybeValid(t *testing.T) {
	t.Parallel()

	m := MaybeValid("hello")
	assert.True(t, m.Valid)
	assert.Equal(t, "hello", m.Val)

	m2 := MaybeValid(42)
	assert.True(t, m2.Valid)
	assert.Equal(t, 42, m2.Val)
}

func TestNewNullString(t *testing.T) {
	t.Parallel()

	ns := NewNullString("hello")
	assert.True(t, ns.Valid)
	assert.Equal(t, "hello", ns.String)

	ns2 := NewNullString("")
	assert.True(t, ns2.Valid)
	assert.Empty(t, ns2.String)
}

func TestNullStringScan(t *testing.T) {
	t.Parallel()

	var ns NullString

	err := ns.Scan("hello")
	require.NoError(t, err)
	assert.True(t, ns.Valid)
	assert.Equal(t, "hello", ns.String)

	var ns2 NullString

	err = ns2.Scan(42)
	require.NoError(t, err)
	assert.False(t, ns2.Valid)
	assert.Empty(t, ns2.String)
}

func TestNullStringValue(t *testing.T) {
	t.Parallel()

	ns := NewNullString("hello")
	v, err := ns.Value()
	require.NoError(t, err)
	assert.Equal(t, "hello", v)

	ns2 := NullString{}
	v, err = ns2.Value()
	require.NoError(t, err)
	assert.Nil(t, v)
}

func TestNullBoolScan(t *testing.T) {
	t.Parallel()

	var nb NullBool

	err := nb.Scan(true)
	require.NoError(t, err)
	assert.True(t, nb.Valid)
	assert.True(t, nb.Bool)

	var nb2 NullBool

	err = nb2.Scan("bad")
	require.NoError(t, err)
	assert.False(t, nb2.Valid)
}

func TestNullBoolValue(t *testing.T) {
	t.Parallel()

	nb := NewNullBool(true)
	v, err := nb.Value()
	require.NoError(t, err)
	assert.Equal(t, true, v)

	nb2 := NullBool{}
	v, err = nb2.Value()
	require.NoError(t, err)
	assert.Nil(t, v)
}

func TestNullFloat32Scan(t *testing.T) {
	t.Parallel()

	var nf NullFloat32

	err := nf.Scan(float64(3.14))
	require.NoError(t, err)
	assert.True(t, nf.Valid)
	assert.InDelta(t, 3.14, nf.Float32, 0.001)

	var nf2 NullFloat32

	err = nf2.Scan("bad")
	require.NoError(t, err)
	assert.False(t, nf2.Valid)
}

func TestNullFloat32Value(t *testing.T) {
	t.Parallel()

	nf := NewNullFloat32(1.5)
	v, err := nf.Value()
	require.NoError(t, err)
	assert.InDelta(t, 1.5, v, 0.0001)

	nf2 := NullFloat32{}
	v, err = nf2.Value()
	require.NoError(t, err)
	assert.Nil(t, v)
}

func TestNullFloat64Scan(t *testing.T) {
	t.Parallel()

	var nf NullFloat64

	err := nf.Scan(3.14159)
	require.NoError(t, err)
	assert.True(t, nf.Valid)
	assert.InDelta(t, 3.14159, nf.Float64, 0.000001)

	var nf2 NullFloat64

	err = nf2.Scan("bad")
	require.NoError(t, err)
	assert.False(t, nf2.Valid)
}

func TestNullUintScan(t *testing.T) {
	t.Parallel()

	var nu NullUint

	err := nu.Scan(int64(42))
	require.NoError(t, err)
	assert.True(t, nu.Valid)
	assert.Equal(t, uint(42), nu.Uint)

	var nu2 NullUint

	err = nu2.Scan("bad")
	require.NoError(t, err)
	assert.False(t, nu2.Valid)
}

func TestNullUintValue(t *testing.T) {
	t.Parallel()

	nu := NewNullUint(10)
	v, err := nu.Value()
	require.NoError(t, err)
	assert.Equal(t, uint(10), v)

	nu2 := NullUint{}
	v, err = nu2.Value()
	require.NoError(t, err)
	assert.Nil(t, v)
}

func TestContentRef(t *testing.T) {
	t.Parallel()

	c := Content{
		Type:   ContentTypeMovie,
		Source: "tmdb",
		ID:     "123",
	}
	ref := c.Ref()
	assert.Equal(t, ContentTypeMovie, ref.Type)
	assert.Equal(t, "tmdb", ref.Source)
	assert.Equal(t, "123", ref.ID)
}

func TestContentIdentifier(t *testing.T) {
	t.Parallel()

	c := Content{Source: "imdb", ID: "tt1234567"}

	id, ok := c.Identifier("imdb")
	assert.True(t, ok)
	assert.Equal(t, "tt1234567", id)

	_, ok = c.Identifier("tmdb")
	assert.False(t, ok)

	_, ok = c.Identifier("")
	assert.False(t, ok)
}

func TestContentExternalLinks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content Content
		wantLen int
		wantURL string
	}{
		{
			name: "imdb",
			content: Content{
				Type:   ContentTypeMovie,
				Source: "imdb",
				ID:     "tt1234567",
			},
			wantLen: 1,
			wantURL: "https://www.imdb.com/title/tt1234567",
		},
		{
			name: "tmdb movie",
			content: Content{
				Type:   ContentTypeMovie,
				Source: "tmdb",
				ID:     "550",
			},
			wantLen: 1,
			wantURL: "https://www.themoviedb.org/movie/550",
		},
		{
			name: "tmdb tv show",
			content: Content{
				Type:   ContentTypeTvShow,
				Source: "tmdb",
				ID:     "1396",
			},
			wantLen: 1,
			wantURL: "https://www.themoviedb.org/tv/1396",
		},
		{
			name: "tvdb",
			content: Content{
				Type:   ContentTypeTvShow,
				Source: "tvdb",
				ID:     "12345",
			},
			wantLen: 1,
			wantURL: "https://www.thetvdb.com/dereferrer/series/12345",
		},
		{
			name: "unknown source",
			content: Content{
				Type:   ContentTypeMovie,
				Source: "unknown",
				ID:     "123",
			},
			wantLen: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			links := tt.content.ExternalLinks()
			assert.Len(t, links, tt.wantLen)

			if tt.wantLen > 0 {
				assert.Equal(t, tt.wantURL, links[0].URL)
			}
		})
	}
}

func TestContentUpdateTsv(t *testing.T) {
	t.Parallel()

	c := &Content{
		Title: "Test Movie",
		ReleaseDate: Date{
			Year:  2024,
			Month: time.January,
			Day:   15,
		},
	}
	c.UpdateTsv()
	assert.NotNil(t, c.Tsv)
	assert.NotEmpty(t, c.Tsv)
}

func TestContentUpdateTsvNoReleaseDate(t *testing.T) {
	t.Parallel()

	c := &Content{
		Title: "Test Movie",
	}
	c.UpdateTsv()
	assert.NotNil(t, c.Tsv)
	assert.NotEmpty(t, c.Tsv)
}

func TestGetExternalLinkURL(t *testing.T) {
	t.Parallel()

	// Test via ExternalLinks since getExternalLinkURL is unexported
	urls := []struct {
		ct    ContentType
		src   string
		id    string
		want  string
		valid bool
	}{
		{ContentTypeMovie, "imdb", "tt123", "https://www.imdb.com/title/tt123", true},
		{ContentTypeTvShow, "imdb", "tt456", "https://www.imdb.com/title/tt456", true},
		{ContentTypeMovie, "tmdb", "550", "https://www.themoviedb.org/movie/550", true},
		{ContentTypeTvShow, "tmdb", "1396", "https://www.themoviedb.org/tv/1396", true},
		{ContentTypeMusic, "tmdb", "1", "https://www.themoviedb.org/movie/1", true},
		{ContentTypeOther, "tmdb", "2", "https://www.themoviedb.org/movie/2", true},
		{ContentTypeMovie, "tvdb", "series1", "https://www.thetvdb.com/dereferrer/series/series1", true},
		{ContentTypeMovie, "nonexistent", "123", "", false},
	}
	for _, u := range urls {
		c := Content{Type: u.ct, Source: u.src, ID: u.id}

		links := c.ExternalLinks()
		if u.valid {
			require.Len(t, links, 1)
			assert.Equal(t, u.want, links[0].URL)
		} else {
			assert.Empty(t, links)
		}
	}
}

func TestDateIsNil(t *testing.T) {
	t.Parallel()

	assert.True(t, Date{}.IsNil())

	d2 := Date{Year: 2024, Month: time.January, Day: 1}
	assert.False(t, d2.IsNil())
}

func TestDateTime(t *testing.T) {
	t.Parallel()

	d := Date{Year: 2024, Month: time.March, Day: 15}
	ti := d.Time()
	assert.Equal(t, 2024, ti.Year())
	assert.Equal(t, time.Month(3), ti.Month())
	assert.Equal(t, 15, ti.Day())
	assert.Equal(t, 0, ti.Hour())
	assert.Equal(t, 0, ti.Minute())
	assert.Equal(t, 0, ti.Second())
	assert.Equal(t, time.UTC, ti.Location())
}

func TestDateTimeZeroValue(t *testing.T) {
	t.Parallel()

	d := Date{Year: 1, Month: time.January, Day: 1}
	ti := d.Time()
	assert.Equal(t, 1, ti.Nanosecond())
}

func TestDateEndOfDayTime(t *testing.T) {
	t.Parallel()

	d := Date{Year: 2024, Month: time.March, Day: 15}
	ti := d.EndOfDayTime()
	assert.Equal(t, 2024, ti.Year())
	assert.Equal(t, time.Month(3), ti.Month())
	assert.Equal(t, 16, ti.Day())
	assert.Equal(t, 0, ti.Hour())
	assert.Equal(t, 0, ti.Minute())
	assert.Equal(t, 0, ti.Second())
	assert.Equal(t, time.UTC, ti.Location())
}

func TestDateIsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		date  Date
		valid bool
	}{
		{"valid date", Date{Year: 2024, Month: time.March, Day: 15}, true},
		{"valid leap year", Date{Year: 2024, Month: time.February, Day: 29}, true},
		{"invalid leap year", Date{Year: 2023, Month: time.February, Day: 29}, false},
		{"month zero", Date{Year: 2024, Month: 0, Day: 15}, false},
		{"month too high", Date{Year: 2024, Month: 13, Day: 15}, false},
		{"day zero", Date{Year: 2024, Month: time.March, Day: 0}, false},
		{"day too high", Date{Year: 2024, Month: time.March, Day: 32}, false},
		{"year too low", Date{Year: 999, Month: time.March, Day: 15}, false},
		{"year too high", Date{Year: 10000, Month: time.March, Day: 15}, false},
		{"april 31 invalid", Date{Year: 2024, Month: time.April, Day: 31}, false},
		{"zero value", Date{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.valid, tt.date.IsValid())
		})
	}
}

func TestEpisodesAddEpisode(t *testing.T) {
	t.Parallel()

	e := make(Episodes)
	e = e.AddEpisode(1, 1)
	e = e.AddEpisode(1, 2)
	e = e.AddEpisode(2, 5)

	epMap, ok := e[1]
	assert.True(t, ok)
	assert.Contains(t, epMap, 1)
	assert.Contains(t, epMap, 2)
	assert.Len(t, epMap, 2)
}

func TestTorrentFileBaseName(t *testing.T) {
	t.Parallel()

	f := TorrentFile{PathParts: []string{"dir", "file.txt"}}
	assert.Equal(t, "file.txt", f.BaseName())

	f2 := TorrentFile{PathParts: []string{"file.mp4"}}
	assert.Equal(t, "file.mp4", f2.BaseName())

	f3 := TorrentFile{PathParts: []string{}}
	assert.Empty(t, f3.BaseName())
}

func TestTorrentFileFileType(t *testing.T) {
	t.Parallel()

	f := TorrentFile{PathParts: []string{"video.mp4"}}
	ft := f.FileType()
	assert.True(t, ft.Valid)
	assert.Equal(t, FileTypeVideo, ft.FileType)

	f2 := TorrentFile{PathParts: []string{"audio.mp3"}}
	ft2 := f2.FileType()
	assert.True(t, ft2.Valid)
	assert.Equal(t, FileTypeAudio, ft2.FileType)

	f3 := TorrentFile{PathParts: []string{}}
	ft3 := f3.FileType()
	assert.False(t, ft3.Valid)

	f4 := TorrentFile{PathParts: []string{"file.unknown_ext_xyz"}}
	ft4 := f4.FileType()
	assert.False(t, ft4.Valid)
}

func TestFileExtensionFromPath(t *testing.T) {
	t.Parallel()

	ext := FileExtensionFromPath("video.mp4")
	assert.True(t, ext.Valid)
	assert.Equal(t, "mp4", ext.String)

	ext = FileExtensionFromPath("/path/to/file.mp4")
	assert.True(t, ext.Valid)
	assert.Equal(t, "mp4", ext.String)

	ext = FileExtensionFromPath("noext")
	assert.False(t, ext.Valid)

	ext = FileExtensionFromPath(".hidden")
	assert.False(t, ext.Valid)

	ext = FileExtensionFromPath("UPPER.MP4")
	assert.True(t, ext.Valid)
	assert.Equal(t, "mp4", ext.String)

	ext = FileExtensionFromPath("archive.tar.gz")
	assert.True(t, ext.Valid)
	assert.Equal(t, "gz", ext.String)
}

func TestFileTypeFromPath(t *testing.T) {
	t.Parallel()

	ft := fileTypeFromPath("video.mp4")
	assert.True(t, ft.Valid)
	assert.Equal(t, FileTypeVideo, ft.FileType)

	ft = fileTypeFromPath("file.unknown")
	assert.False(t, ft.Valid)

	ft = fileTypeFromPath("noext")
	assert.False(t, ft.Valid)
}

func TestTorrentBaseName(t *testing.T) {
	t.Parallel()

	tor := Torrent{Name: "test-video.mp4"}
	assert.Equal(t, "test-video", tor.BaseName())

	tor2 := Torrent{Name: "no-extension"}
	assert.Equal(t, "no-extension", tor2.BaseName())

	tor3 := Torrent{Name: ".hidden"}
	assert.Equal(t, ".hidden", tor3.BaseName())
}

func TestTorrentSingleFile(t *testing.T) {
	t.Parallel()

	tor := Torrent{FilesCount: NullUint{Uint: 1, Valid: true}}
	assert.True(t, tor.SingleFile())

	tor2 := Torrent{FilesCount: NullUint{Uint: 5, Valid: true}}
	assert.False(t, tor2.SingleFile())

	tor3 := Torrent{FilesCount: NullUint{Valid: false}}
	assert.False(t, tor3.SingleFile())

	tor4 := Torrent{FilesCount: NullUint{Uint: 0, Valid: true}}
	assert.False(t, tor4.SingleFile())
}

func TestTorrentHasFilesInfo(t *testing.T) {
	t.Parallel()

	tor := Torrent{FilesCount: NullUint{Uint: 5, Valid: true}}
	assert.True(t, tor.HasFilesInfo())

	tor2 := Torrent{Files: []TorrentFile{{InfoHash: protocol.ID{}}}}
	assert.True(t, tor2.HasFilesInfo())

	tor3 := Torrent{}
	assert.False(t, tor3.HasFilesInfo())

	tor4 := Torrent{FilesCount: NullUint{Valid: false}}
	assert.False(t, tor4.HasFilesInfo())
}

func TestTorrentMagnetURI(t *testing.T) {
	t.Parallel()

	id := protocol.ID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
	tor := Torrent{
		InfoHash: id,
		Name:     "test torrent",
		Size:     1024,
	}
	uri := tor.MagnetURI()
	assert.Contains(t, uri, "magnet:?xt=urn:btih:")
	assert.Contains(t, uri, id.String())
	assert.Contains(t, uri, "&dn=test+torrent")
	assert.Contains(t, uri, "&xl=1024")
}

func TestTorrentFileType(t *testing.T) {
	t.Parallel()

	tor := Torrent{Name: "video.mp4"}
	ft := tor.FileType()
	assert.True(t, ft.Valid)
	assert.Equal(t, FileTypeVideo, ft.FileType)

	tor2 := Torrent{Name: "noext"}
	ft2 := tor2.FileType()
	assert.False(t, ft2.Valid)
}

func TestTorrentFileExtensions(t *testing.T) {
	t.Parallel()

	tor := Torrent{
		Name:       "archive.rar",
		FilesCount: NullUint{Uint: 1, Valid: true},
	}
	exts := tor.FileExtensions()
	assert.Contains(t, exts, "rar")

	tor2 := Torrent{
		Name: "multi-file",
		Files: []TorrentFile{
			{PathParts: []string{"video.mp4"}},
			{PathParts: []string{"subs.srt"}},
		},
	}
	exts2 := tor2.FileExtensions()
	assert.Contains(t, exts2, "mp4")
	assert.Contains(t, exts2, "srt")
	assert.Len(t, exts2, 2)

	tor3 := Torrent{
		Name:       "single.mp4",
		FilesCount: NullUint{Uint: 1, Valid: true},
		Files: []TorrentFile{
			{PathParts: []string{"single.mp4"}},
		},
	}
	exts3 := tor3.FileExtensions()
	assert.Contains(t, exts3, "mp4")
}

func TestTorrentFileExtensionsDeduplicates(t *testing.T) {
	t.Parallel()

	tor := Torrent{
		Name: "multi",
		Files: []TorrentFile{
			{PathParts: []string{"a.mp4"}},
			{PathParts: []string{"b.mp4"}},
		},
	}
	exts := tor.FileExtensions()
	assert.Contains(t, exts, "mp4")
	assert.Len(t, exts, 1)
}

func TestTorrentFileExtensionsEmpty(t *testing.T) {
	t.Parallel()

	tor := Torrent{Name: "noext"}
	exts := tor.FileExtensions()
	assert.Empty(t, exts)
}

func TestYearString(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "2024", Year(2024).String())
	assert.Equal(t, "0", Year(0).String())
	assert.Equal(t, "1999", Year(1999).String())
}

func TestYearIsNil(t *testing.T) {
	t.Parallel()

	assert.True(t, Year(0).IsNil())
	assert.False(t, Year(2024).IsNil())
}

func TestYearScan(t *testing.T) {
	t.Parallel()

	var y Year

	err := y.Scan(2024)
	require.NoError(t, err)
	assert.Equal(t, Year(2024), y)

	var y2 Year

	err = y2.Scan(int32(2023))
	require.NoError(t, err)
	assert.Equal(t, Year(2023), y2)

	var y3 Year

	err = y3.Scan(int64(2022))
	require.NoError(t, err)
	assert.Equal(t, Year(2022), y3)

	var y4 Year

	err = y4.Scan(uint(2021))
	require.NoError(t, err)
	assert.Equal(t, Year(2021), y4)

	var y5 Year

	err = y5.Scan(float64(2020.0))
	require.NoError(t, err)
	assert.Equal(t, Year(2020), y5)

	var y6 Year

	err = y6.Scan(nil)
	require.NoError(t, err)
	assert.Equal(t, Year(0), y6)

	var y7 Year

	err = y7.Scan("1999")
	require.NoError(t, err)
	assert.Equal(t, Year(1999), y7)

	var y8 Year

	err = y8.Scan("not-a-year")
	require.Error(t, err)

	var y9 Year

	err = y9.Scan(true)
	assert.Error(t, err)
}

func TestYearValue(t *testing.T) {
	t.Parallel()

	y := Year(2024)
	v, err := y.Value()
	require.NoError(t, err)
	assert.Equal(t, 2024, v)

	y2 := Year(0)
	v, err = y2.Value()
	require.NoError(t, err)
	assert.Nil(t, v)
}

func TestParseYear(t *testing.T) {
	t.Parallel()

	y, err := ParseYear("2024")
	require.NoError(t, err)
	assert.Equal(t, Year(2024), y)

	_, err = ParseYear("not-a-year")
	require.Error(t, err)

	_, err = ParseYear("")
	require.Error(t, err)

	y, err = ParseYear("1999")
	require.NoError(t, err)
	assert.Equal(t, Year(1999), y)
}

func TestDateScan(t *testing.T) {
	t.Parallel()

	var d Date

	err := d.Scan(time.Date(2024, time.March, 15, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	assert.Equal(t, Year(2024), d.Year)
	assert.Equal(t, time.March, d.Month)
	assert.Equal(t, uint8(15), d.Day)

	var d2 Date

	err = d2.Scan(time.Time{})
	require.NoError(t, err)
	assert.True(t, d2.IsNil())

	var d3 Date

	err = d3.Scan("not-a-time")
	require.NoError(t, err)
	assert.True(t, d3.IsNil())
}

func TestDateValue(t *testing.T) {
	t.Parallel()

	d := Date{Year: 2024, Month: time.March, Day: 15}
	v, err := d.Value()
	require.NoError(t, err)

	ti, ok := v.(time.Time)
	require.True(t, ok)
	assert.Equal(t, 2024, ti.Year())

	d2 := Date{}
	v, err = d2.Value()
	require.NoError(t, err)
	assert.Nil(t, v)
}
