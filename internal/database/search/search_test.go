package dbsearch

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewFromPool(t *testing.T) {
	t.Parallel()

	s := NewFromPool(nil)
	assert.NotNil(t, s)
	_, ok := s.(*pgSearch)
	assert.True(t, ok)
}

func TestFileTypeExtensions_Video(t *testing.T) {
	t.Parallel()

	expected := []string{"avi", "divx", "m2ts", "m4v", "mkv", "mov", "mp4", "mpeg", "mpg", "ts", "webm", "wmv", "vob", "iso"}
	assert.Equal(t, expected, fileTypeExtensions("video"))
}

func TestFileTypeExtensions_Audio(t *testing.T) {
	t.Parallel()

	expected := []string{"aac", "ac3", "ape", "dts", "flac", "m4a", "mp3", "ogg", "opus", "wav", "wma"}
	assert.Equal(t, expected, fileTypeExtensions("audio"))
}

func TestFileTypeExtensions_Image(t *testing.T) {
	t.Parallel()

	expected := []string{"bmp", "gif", "jpg", "jpeg", "png", "tbn", "webp"}
	assert.Equal(t, expected, fileTypeExtensions("image"))
}

func TestFileTypeExtensions_Archive(t *testing.T) {
	t.Parallel()

	expected := []string{"7z", "bz2", "gz", "lzma", "rar", "tar", "xz", "zst", "zip", "zpaq"}
	assert.Equal(t, expected, fileTypeExtensions("archive"))
}

func TestFileTypeExtensions_Subtitle(t *testing.T) {
	t.Parallel()

	expected := []string{"ass", "idx", "srt", "ssa", "sub", "sup", "vtt"}
	assert.Equal(t, expected, fileTypeExtensions("subtitle"))
}

func TestFileTypeExtensions_Document(t *testing.T) {
	t.Parallel()

	expected := []string{"doc", "docx", "htm", "html", "mht", "mobi", "pdf", "txt", "xls", "xlsx", "xml"}
	assert.Equal(t, expected, fileTypeExtensions("document"))
}

func TestFileTypeExtensions_Metadata(t *testing.T) {
	t.Parallel()

	expected := []string{"nfo", "sfv"}
	assert.Equal(t, expected, fileTypeExtensions("metadata"))
}

func TestFileTypeExtensions_Unknown(t *testing.T) {
	t.Parallel()

	assert.Nil(t, fileTypeExtensions("unknown"))
	assert.Nil(t, fileTypeExtensions(""))
}
