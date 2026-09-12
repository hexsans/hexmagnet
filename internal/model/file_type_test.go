package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileType_Extensions(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []string{"mkv", "mp4"}, intersect(FileTypeVideo.Extensions(), []string{"mkv", "mp4"}))
	assert.Contains(t, FileTypeSubtitles.Extensions(), "srt")
	assert.Empty(t, FileType("unknown").Extensions())
}

func TestFileType_ExtensionsRoundTrip(t *testing.T) {
	t.Parallel()

	for _, ft := range FileTypeValues() {
		exts := ft.Extensions()
		require.NotEmpty(t, exts, "file type %q should have extensions", ft)

		for _, ext := range exts {
			mapped := FileTypeFromExtension(ext)
			require.True(t, mapped.Valid, "extension %q should be mapped", ext)
			assert.Equal(t, ft, mapped.FileType, "extension %q", ext)
		}
	}
}

func intersect(a, b []string) []string {
	set := make(map[string]struct{}, len(a))
	for _, v := range a {
		set[v] = struct{}{}
	}

	result := make([]string, 0, len(b))

	for _, v := range b {
		if _, ok := set[v]; ok {
			result = append(result, v)
		}
	}

	return result
}
