package classifier

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildPrompt_NoYearField(t *testing.T) {
	t.Parallel()

	systemMsg, _ := BuildPrompt("Test Movie 2024", nil, "", 0)

	require.NotContains(t, systemMsg, `"year"`, "prompt should not contain year field")
	require.Contains(t, systemMsg, `"date"`, "prompt should contain date field")
	require.Contains(t, systemMsg, `"YYYY"`, "prompt should mention YYYY fallback format")
}

func TestBuildPrompt_ReasoningEffortNone(t *testing.T) {
	t.Parallel()

	systemMsgNone, _ := BuildPrompt("Test", nil, "none", 0)
	require.Contains(t, systemMsgNone, "DIRECT INSTRUCTION",
		"prompt should contain DIRECT INSTRUCTION when reasoningEffort is none")

	systemMsgLow, _ := BuildPrompt("Test", nil, "low", 0)
	require.NotContains(t, systemMsgLow, "DIRECT INSTRUCTION",
		"prompt should not contain DIRECT INSTRUCTION when reasoningEffort is not none")

	systemMsgEmpty, _ := BuildPrompt("Test", nil, "", 0)
	require.NotContains(t, systemMsgEmpty, "DIRECT INSTRUCTION",
		"prompt should not contain DIRECT INSTRUCTION when reasoningEffort is empty")
}

func TestBuildPrompt_IncludesTorrentName(t *testing.T) {
	t.Parallel()

	systemMsg, userMsg := BuildPrompt("Test Movie 2024", nil, "", 0)

	require.Contains(t, systemMsg, "You are a torrent classifier")
	require.Contains(t, userMsg, "Torrent name: Test Movie 2024")
}

func TestBuildPrompt_IncludesFiles(t *testing.T) {
	t.Parallel()

	files := []TorrentFile{
		{Path: "video.mkv", Size: 1024, Extension: "mkv"},
		{Path: "subtitle.srt", Size: 512, Extension: "srt"},
	}
	_, userMsg := BuildPrompt("Test", files, "", 10)

	require.Contains(t, userMsg, "Files:")
	require.Contains(t, userMsg, "video.mkv")
	require.Contains(t, userMsg, "subtitle.srt")
}

func TestBuildPrompt_SortsFilesBySize(t *testing.T) {
	t.Parallel()

	files := []TorrentFile{
		{Path: "small.txt", Size: 100, Extension: "txt"},
		{Path: "large.mkv", Size: 10000, Extension: "mkv"},
	}
	_, userMsg := BuildPrompt("Test", files, "", 10)

	largeIdx := strings.Index(userMsg, "large.mkv")
	smallIdx := strings.Index(userMsg, "small.txt")
	require.Less(t, largeIdx, smallIdx, "larger file should appear first")
}
