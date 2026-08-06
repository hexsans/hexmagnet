package classifier

import (
	"strings"

	"github.com/hexsans/hexmagnet/internal/model"
)

func NewTorrentFromModel(t model.Torrent) map[string]any {
	var (
		files     []any
		filesSize *int64
	)

	switch {
	case !t.FilesCount.Valid:
		ext := model.FileExtensionFromPath(t.Name)

		f := map[string]any{
			celFieldPath:      t.Name,
			celFieldBasePath:  t.Name,
			celFieldBaseName:  t.Name,
			celFieldSize:      int64(t.Size),
			celFieldExtension: nullableString(ext.String, ext.Valid),
			celFieldFileType:  int32(FileTypeUnknown),
		}
		if ext.Valid {
			f["fileType"] = FileTypeToInt(model.FileTypeFromExtension(ext.String))
		}

		files = append(files, f)
		s := int64(t.Size)
		filesSize = &s

	case t.FilesCount.Uint == 1:
		ft := t.FileType()
		f := map[string]any{
			celFieldPath:     t.Name,
			celFieldBasePath: t.BaseName(),
			celFieldBaseName: t.BaseName(),
			celFieldSize:     int64(t.Size),
			celFieldExtension: nullableString(
				model.FileExtensionFromPath(t.Name).String,
				model.FileExtensionFromPath(t.Name).Valid,
			),
			celFieldFileType: FileTypeToInt(ft),
		}
		files = append(files, f)
		s := int64(t.Size)
		filesSize = &s

	default:
		var fs int64
		for _, f := range t.Files {
			fs += int64(f.Size)
			files = append(files, map[string]any{
				"index":           int32(f.Index),
				celFieldPath:      strings.Join(f.PathParts, "/"),
				celFieldBasePath:  f.BasePath(),
				celFieldBaseName:  f.BaseName(),
				celFieldSize:      int64(f.Size),
				celFieldExtension: nullableString(f.Extension.String, f.Extension.Valid),
				celFieldFileType:  FileTypeToInt(f.FileType()),
			})
		}

		filesSize = &fs
	}

	m := map[string]any{
		"infoHash":        t.InfoHash.String(),
		"name":            t.Name,
		celFieldBaseName:  t.BaseName(),
		celFieldSize:      int64(t.Size),
		celFieldExtension: nullableString(model.FileExtensionFromPath(t.Name).String, model.FileExtensionFromPath(t.Name).Valid),
		"files":           files,
		"filesCount":      nullableUint(t.FilesCount.Uint, t.FilesCount.Valid),
		"filesSize":       filesSize,
		"fileExtensions":  t.FileExtensions(),
	}

	return m
}
