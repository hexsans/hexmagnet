package classifier

import (
	"fmt"
	"sort"
	"strings"
)

func BuildPrompt(name string, files []TorrentFile, reasoningEffort string, maxFiles int) (systemMsg, userMsg string) {
	systemMsg = `You are a torrent classifier. Output ONLY valid JSON with these fields:
- "type": "movie"|"tv_show"|"music"|"ebook"|"comic"|"audiobook"|"game"|"software"|"adult"|"other"|"unknown"
- "base_title": clean core content name only (no year, tags, group, episode, resolution, codec, URLs, metadata). ` +
		`JAV content uses the JAV ID (e.g. "XXX-000"). Music uses "Artist - Album".
- "date": "2006-01-02" ISO format, or just "YYYY" if month/day unclear; omit if unclear
- "languages": ISO 639-1 codes array; omit if unclear

Classify using file extensions (.mp4/.mkv = video, .mp3/.flac = audio, .epub/.mobi = ebook, ` +
		`.cbz/.cbr = comic, .iso/.nsp/.xci = game) and name patterns (S##E## = tv_show, ` +
		`year in parentheses = movie, JAV ID XXX-000 = adult). Omit any field that is not applicable. ` +
		`Return ONLY valid JSON, no other text.`

	if reasoningEffort == reasoningEffortNone {
		systemMsg += "\n\nDIRECT INSTRUCTION: Do not use any internal reasoning, chain-of-thought, thinking tags " +
			`(like <think>, <reasoning>), or any thought process. Output the JSON object immediately as the first ` +
			`and only token of your response. Your response must begin with { and contain nothing else.`
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
		_, _ = fmt.Fprintf(&b, "  - %s\n", f.Path)
	}

	return systemMsg, b.String()
}
