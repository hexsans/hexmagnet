package classifier

import (
	"fmt"
	"sort"
	"strings"
)

//nolint:revive // the prompt is intentionally kept as a single readable raw string
const defaultSystemPrompt = `You are a torrent classifier. Output one compact single-line JSON object and nothing else:
- "type": "movie"|"tv_show"|"music"|"ebook"|"comic"|"audiobook"|"game"|"software"|"adult"|"other"|"unknown"
- "base_title": core name only (no year, tags, group, episode, resolution, codec, URL); music as "Artist - Album"; JAV as its ID (e.g. "ABC-123")
- "date" (nullable): content release/first-air date; use "YYYY-MM-DD" when day and month are explicit, otherwise "YYYY" when only the year is explicit. Only use a date explicitly written in the torrent name or file names (e.g. "2021-06-19", "23.02.01", "Jan 1 2020", "(2021)"). Never infer or decode dates from content IDs, catalog knowledge, or compact digit runs (e.g. "FC2-PPV-1485672", "012514"); null if not explicit
- "languages" (nullable): content audio/text languages as ISO 639-1 codes (not subtitle-only); null if unclear

Use null for unknown "date"/"languages"; never emit "" or []. Use signals in this order: (1) torrent name and file names, (2) file extensions. Name patterns: S##E##/1x02/Season = tv_show, "(year)" = movie, "Artist - Album" = music, JAV ID = adult. Extension hints: .mkv/.mp4/.avi/.ts video, .mp3/.flac/.m4a music, .m4b/.aax audiobook, .epub/.mobi/.azw3/.pdf ebook, .cbz/.cbr comic, .iso/.nsp/.xci/.rom game, .exe/.msi/.dmg/.apk software. When they conflict, follow the names; extensions only confirm. If unsure use "unknown". No markdown fences.`

// DefaultSystemPrompt returns the built-in system prompt used when no custom
// prompt is configured.
func DefaultSystemPrompt() string {
	return defaultSystemPrompt
}

func BuildPrompt(
	name, customPrompt string,
	files []TorrentFile,
	maxFiles int,
) (systemMsg, userMsg string) {
	systemMsg = defaultSystemPrompt
	if customPrompt = strings.TrimSpace(customPrompt); customPrompt != "" {
		systemMsg = customPrompt
	}

	var b strings.Builder

	_, _ = fmt.Fprintf(&b, "Torrent name: %s\n\n", name)

	if maxFiles == 0 {
		return systemMsg, b.String()
	}

	sorted := make([]TorrentFile, len(files))
	copy(sorted, files)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Size > sorted[j].Size
	})

	if maxFiles > 0 && len(sorted) > maxFiles {
		sorted = sorted[:maxFiles]
	}

	_, _ = b.WriteString("Files:\n")

	for _, f := range sorted {
		_, _ = fmt.Fprintf(&b, "  - %s (%s)\n", f.Path, humanReadableSize(f.Size))
	}

	return systemMsg, b.String()
}

func humanReadableSize(size uint64) string {
	const unit = 1000

	if size < unit {
		return fmt.Sprintf("%d B", size)
	}

	div, exp := uint64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "kMGTPE"[exp])
}
