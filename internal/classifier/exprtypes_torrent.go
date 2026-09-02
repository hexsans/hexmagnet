package classifier

import (
	"strings"

	"github.com/hexsans/hexmagnet/internal/model"
)

type Torrent struct {
	InfoHash       string   `expr:"infoHash"`
	Name           string   `expr:"name"`
	BaseName       string   `expr:"baseName"`
	Size           int64    `expr:"size"`
	Extension      string   `expr:"extension"`
	Files          []File   `expr:"files"`
	FilesCount     uint64   `expr:"filesCount"`
	FilesSize      int64    `expr:"filesSize"`
	FileExtensions []string `expr:"fileExtensions"`
}

type File struct {
	Index     int32  `expr:"index"`
	Path      string `expr:"path"`
	BasePath  string `expr:"basePath"`
	BaseName  string `expr:"baseName"`
	Size      int64  `expr:"size"`
	Extension string `expr:"extension"`
	FileType  int32  `expr:"fileType"`
}

func NewTorrentFromModel(t model.Torrent) Torrent {
	var (
		files      []File
		filesSize  int64
		filesCount uint64
	)

	switch {
	case !t.FilesCount.Valid:
		ext := model.FileExtensionFromPath(t.Name)

		files = append(files, File{
			Path:      t.Name,
			BasePath:  t.Name,
			BaseName:  t.Name,
			Size:      int64(t.Size),
			Extension: ext.String,
			FileType:  FileTypeToInt(model.FileTypeFromExtension(ext.String)),
		})
		filesSize = int64(t.Size)

	case t.FilesCount.Uint == 1:
		ft := t.FileType()
		ext := model.FileExtensionFromPath(t.Name)

		files = append(files, File{
			Path:      t.Name,
			BasePath:  t.BaseName(),
			BaseName:  t.BaseName(),
			Size:      int64(t.Size),
			Extension: ext.String,
			FileType:  FileTypeToInt(ft),
		})
		filesSize = int64(t.Size)
		filesCount = 1

	default:
		for _, f := range t.Files {
			filesSize += int64(f.Size)
			files = append(files, File{
				Index:     int32(f.Index),
				Path:      strings.Join(f.PathParts, "/"),
				BasePath:  f.BasePath(),
				BaseName:  f.BaseName(),
				Size:      int64(f.Size),
				Extension: f.Extension.String,
				FileType:  FileTypeToInt(f.FileType()),
			})
		}

		filesCount = uint64(t.FilesCount.Uint)
	}

	return Torrent{
		InfoHash:       t.InfoHash.String(),
		Name:           t.Name,
		BaseName:       t.BaseName(),
		Size:           int64(t.Size),
		Extension:      model.FileExtensionFromPath(t.Name).String,
		Files:          files,
		FilesCount:     filesCount,
		FilesSize:      filesSize,
		FileExtensions: t.FileExtensions(),
	}
}
