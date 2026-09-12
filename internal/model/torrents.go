package model

import (
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/hexsans/hexmagnet/internal/database/fts"
)

func (t Torrent) MagnetURI() string {
	return "magnet:?xt=urn:btih:" + t.InfoHash.String() +
		"&dn=" + url.QueryEscape(t.Name) +
		"&xl=" + strconv.FormatUint(t.Size, 10)
}

// HasFilesInfo returns true if we know about the files in this torrent.
func (t Torrent) HasFilesInfo() bool {
	return t.FilesCount.Valid || len(t.Files) > 0
}

func (t Torrent) SingleFile() bool {
	return t.FilesCount.Valid && t.FilesCount.Uint == 1
}

func (t Torrent) BaseName() string {
	ext := FileExtensionFromPath(t.Name)
	if ext.Valid {
		return t.Name[:len(t.Name)-len(ext.String)-1]
	}

	return t.Name
}

func (t Torrent) FileExtensions() []string {
	exts := make([]string, 0, max(1, len(t.Files)))
	extMap := make(map[string]struct{})

	extract := func(path string) {
		ext := FileExtensionFromPath(path)
		if ext.Valid {
			if _, ok := extMap[ext.String]; !ok {
				extMap[ext.String] = struct{}{}
				exts = append(exts, ext.String)
			}
		}
	}

	if t.FilesCount.Valid && t.FilesCount.Uint == 1 && len(t.Files) == 0 {
		extract(t.Name)
	} else {
		for _, file := range t.Files {
			extract(strings.Join(file.PathParts, "/"))
		}
	}

	return exts
}

func (t Torrent) FileType() NullFileType {
	ext := FileExtensionFromPath(t.Name)
	if ext.Valid {
		return FileTypeFromExtension(ext.String)
	}

	return NullFileType{}
}

func (t Torrent) FileTypes() []FileType {
	exts := t.FileExtensions()
	typesMap := make(map[FileType]struct{})
	types := make([]FileType, 0, len(exts))

	for _, ext := range exts {
		if ft := FileTypeFromExtension(ext); ft.Valid {
			if _, ok := typesMap[ft.FileType]; !ok {
				typesMap[ft.FileType] = struct{}{}

				types = append(types, ft.FileType)
			}
		}
	}

	return types
}

func (t Torrent) HasFileType(fts ...FileType) NullBool {
	for _, thisFt := range t.FileTypes() {
		if slices.Contains(fts, thisFt) {
			return NewNullBool(true)
		}
	}

	return NewNullBool(false)
}

// fileSearchStrings returns a list of strings extracted from file paths, for inclusion in the text search vector.
// To reduce duplication, common prefixes and suffixes are deduplicated.
// maxFiles bounds how many file entries are considered: large torrents (thousands
// of files) would otherwise produce enormous tsvectors. A value <= 0 means no cap.
func (t Torrent) fileSearchStrings(maxFiles int) []string {
	if maxFiles > 0 && len(t.Files) > maxFiles {
		t.Files = t.Files[:maxFiles]
	}

	firstPass := make([]string, 0, len(t.Files))

	var prevPath string

outer:
	for _, f := range t.Files {
		fp := strings.Join(f.PathParts, "/")

		i := 0
		for {
			if i >= len(fp) {
				continue outer
			}

			if i >= len(prevPath) || prevPath[i] != fp[i] {
				break
			}

			i++
		}

		for i != 0 && fts.IsWordChar(rune(fp[i])) {
			i--
		}

		firstPass = append(firstPass, fp[i:])
		prevPath = fp
	}

	searchStrings := make([]string, 0, len(firstPass))

	for i := range firstPass {
		longestSuffixLength := 0

		for j := range i {
			l := 0

			for l < len(firstPass[i]) &&
				l < len(firstPass[j]) &&
				firstPass[i][len(firstPass[i])-l-1] == firstPass[j][len(firstPass[j])-l-1] {
				l++
			}

			if l > longestSuffixLength {
				longestSuffixLength = l
			}
		}

		for longestSuffixLength != 0 &&
			fts.IsWordChar(rune(firstPass[i][len(firstPass[i])-longestSuffixLength])) {
			longestSuffixLength--
		}

		str := strings.TrimSpace(firstPass[i][:len(firstPass[i])-longestSuffixLength])
		if str != "" {
			searchStrings = append(searchStrings, str)
		}
	}

	return searchStrings
}

func (t Torrent) InferID() string {
	parts := make([]string, 4)
	parts[0] = t.InfoHash.String()

	if t.ContentType.Valid {
		parts[1] = t.ContentType.ContentType.String()
	} else {
		parts[1] = "?"
	}

	if t.ContentSource.Valid {
		parts[2] = t.ContentSource.String
		parts[3] = t.ContentID.String
	} else {
		parts[2] = "?"
		parts[3] = "?"
	}

	return strings.Join(parts, ":")
}

func (t Torrent) Title() string {
	if !t.ContentID.Valid || t.Content.Title == "" {
		return t.Name
	}

	var titleParts []string

	titleParts = append(titleParts, t.Content.Title)

	if !t.Content.ReleaseDate.IsNil() {
		titleParts = append(titleParts, fmt.Sprintf("(%d)", t.Content.ReleaseDate.Year))
	}

	return strings.Join(titleParts, " ")
}

func (t Torrent) ContentRef() Maybe[ContentRef] {
	if t.ContentID.Valid {
		return MaybeValid(ContentRef{
			Type:   t.ContentType.ContentType,
			Source: t.ContentSource.String,
			ID:     t.ContentID.String,
		})
	}

	return Maybe[ContentRef]{}
}

// UpdateTsv rebuilds the full-text search vector for a torrent. maxSearchFiles
// bounds the number of file paths included (in addition to name and content
// metadata); pass a value <= 0 for no cap.
func (t *Torrent) UpdateTsv(maxSearchFiles int) {
	var tsv fts.Tsvector
	if !t.ContentID.Valid {
		tsv = fts.Tsvector{}
	} else {
		tsv = t.Content.Tsv.Copy()
	}

	tsv.AddText(t.Name, fts.TsvectorWeightA)

	for _, str := range t.fileSearchStrings(maxSearchFiles) {
		tsv.AddText(str, fts.TsvectorWeightD)
	}

	t.Tsv = tsv
}
