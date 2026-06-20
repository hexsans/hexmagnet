package classifier

import (
	"github.com/hexsans/hexmagnet/internal/model"
	"github.com/hexsans/hexmagnet/internal/utils"
)

const (
	FileTypeUnknown   = 0
	FileTypeArchive   = 1
	FileTypeAudio     = 2
	FileTypeData      = 3
	FileTypeDocument  = 4
	FileTypeImage     = 5
	FileTypeSoftware  = 6
	FileTypeSubtitles = 7
	FileTypeVideo     = 8

	ContentTypeUnknown   = 0
	ContentTypeMovie     = 1
	ContentTypeTvShow    = 2
	ContentTypeMusic     = 3
	ContentTypeEbook     = 4
	ContentTypeComic     = 5
	ContentTypeAudiobook = 6
	ContentTypeGame      = 7
	ContentTypeSoftware  = 8
	ContentTypeOther     = 9
	ContentTypeAdult     = 10
)

func NewClassificationFromResult(r ClassificationResult) map[string]any {
	var (
		contentID     *string
		contentSource *string
	)

	if r.Content != nil {
		contentID = &r.Content.ID
		contentSource = &r.Content.Source
	}

	var dateVal int64
	if !r.Date.IsNil() {
		dateVal = r.Date.Time().Unix()
	}

	return map[string]any{
		"contentType":        ContentTypeToInt(r.ContentType),
		"hasAttachedContent": r.Content != nil,
		"hasBaseTitle":       r.BaseTitle.Valid,
		"date":               dateVal,
		"languages":          utils.Map(r.Languages.Slice(), func(l model.Language) string { return l.ID() }),
		"contentId":          nullableStringPtr(contentID),
		"contentSource":      nullableStringPtr(contentSource),
	}
}

func nullableStringPtr(s *string) any {
	if s == nil {
		return nil
	}

	return *s
}

func FileTypeToInt(ft model.NullFileType) int32 {
	if !ft.Valid {
		return FileTypeUnknown
	}

	switch ft.FileType {
	case model.FileTypeArchive:
		return FileTypeArchive
	case model.FileTypeAudio:
		return FileTypeAudio
	case model.FileTypeData:
		return FileTypeData
	case model.FileTypeDocument:
		return FileTypeDocument
	case model.FileTypeImage:
		return FileTypeImage
	case model.FileTypeSoftware:
		return FileTypeSoftware
	case model.FileTypeSubtitles:
		return FileTypeSubtitles
	case model.FileTypeVideo:
		return FileTypeVideo
	default:
		return FileTypeUnknown
	}
}

func ContentTypeToInt(ct model.NullContentType) int32 {
	if !ct.Valid {
		return ContentTypeUnknown
	}

	switch ct.ContentType {
	case model.ContentTypeMovie:
		return ContentTypeMovie
	case model.ContentTypeTvShow:
		return ContentTypeTvShow
	case model.ContentTypeMusic:
		return ContentTypeMusic
	case model.ContentTypeEbook:
		return ContentTypeEbook
	case model.ContentTypeComic:
		return ContentTypeComic
	case model.ContentTypeAudiobook:
		return ContentTypeAudiobook
	case model.ContentTypeGame:
		return ContentTypeGame
	case model.ContentTypeSoftware:
		return ContentTypeSoftware
	case model.ContentTypeOther:
		return ContentTypeOther
	case model.ContentTypeAdult:
		return ContentTypeAdult
	default:
		return ContentTypeUnknown
	}
}
