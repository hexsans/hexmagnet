package classifier

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildPrompt_NoYearField(t *testing.T) {
	t.Parallel()

	systemMsg, _ := BuildPrompt("Test Movie 2024", "", nil, 0)

	require.NotContains(t, systemMsg, `"year"`, "prompt should not contain year field")
	require.Contains(t, systemMsg, `"date"`, "prompt should contain date field")
	require.Contains(t, systemMsg, `"YYYY"`, "prompt should mention YYYY fallback format")
}

func TestBuildPrompt_IncludesTorrentName(t *testing.T) {
	t.Parallel()

	systemMsg, userMsg := BuildPrompt("Test Movie 2024", "", nil, 0)

	require.Contains(t, systemMsg, "You are a torrent classifier")
	require.Contains(t, userMsg, "Torrent name: Test Movie 2024")
}

func TestBuildPrompt_DefaultIsCompactJSON(t *testing.T) {
	t.Parallel()

	systemMsg, _ := BuildPrompt("Test", "", nil, 0)

	require.Contains(t, systemMsg, "compact single-line JSON")
	require.Contains(t, systemMsg, "No markdown fences")
	require.Contains(t, systemMsg, `"base_title"`)
}

func TestBuildPrompt_CustomPrompt(t *testing.T) {
	t.Parallel()

	systemMsg, userMsg := BuildPrompt("Test Movie 2024", "Custom classifier instructions", nil, 0)
	require.Equal(t, "Custom classifier instructions", systemMsg)
	require.Contains(t, userMsg, "Torrent name: Test Movie 2024")

	blankMsg, _ := BuildPrompt("Test", "   \n\t ", nil, 0)
	require.Contains(t, blankMsg, "You are a torrent classifier",
		"blank custom prompt should fall back to the default")
}

func TestBuildPrompt_IncludesFiles(t *testing.T) {
	t.Parallel()

	files := []TorrentFile{
		{Path: "video.mkv", Size: 1024, Extension: "mkv"},
		{Path: "subtitle.srt", Size: 512, Extension: "srt"},
	}
	_, userMsg := BuildPrompt("Test", "", files, 10)

	require.Contains(t, userMsg, "Files:")
	require.Contains(t, userMsg, "video.mkv (1.0 kB)")
	require.Contains(t, userMsg, "subtitle.srt (512 B)")
}

func TestBuildPrompt_SortsFilesBySize(t *testing.T) {
	t.Parallel()

	files := []TorrentFile{
		{Path: "small.txt", Size: 100, Extension: "txt"},
		{Path: "large.mkv", Size: 10000, Extension: "mkv"},
	}
	_, userMsg := BuildPrompt("Test", "", files, 10)

	largeIdx := strings.Index(userMsg, "large.mkv")
	smallIdx := strings.Index(userMsg, "small.txt")
	require.Less(t, largeIdx, smallIdx, "larger file should appear first")
}
