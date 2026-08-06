package model

import (
	"regexp"
	"strings"
)

func (f TorrentFile) BasePath() string {
	if len(f.PathParts) == 0 {
		return ""
	}

	baseName := strings.Join(f.PathParts, "/")
	if f.Extension.Valid {
		baseName = baseName[:len(baseName)-len(f.Extension.String)-1]
	}

	return baseName
}

func (f TorrentFile) BaseName() string {
	if len(f.PathParts) == 0 {
		return ""
	}

	return f.PathParts[len(f.PathParts)-1]
}

var fileExtensionRegex = regexp.MustCompile(`[^/.]\.([a-z0-9]+)$`)

func FileExtensionFromPath(path string) NullString {
	match := fileExtensionRegex.FindStringSubmatch(strings.ToLower(path))
	if len(match) == 2 {
		return NewNullString(match[1])
	}

	return NullString{}
}

func fileTypeFromPath(path string) NullFileType {
	extension := FileExtensionFromPath(path)
	if extension.Valid {
		return FileTypeFromExtension(extension.String)
	}

	return NullFileType{}
}

func (f TorrentFile) FileType() NullFileType {
	if len(f.PathParts) == 0 {
		return NullFileType{}
	}

	return fileTypeFromPath(f.PathParts[len(f.PathParts)-1])
}
