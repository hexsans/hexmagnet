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

type Classification struct {
	ContentType        int32    `expr:"contentType"`
	HasAttachedContent bool     `expr:"hasAttachedContent"`
	HasBaseTitle       bool     `expr:"hasBaseTitle"`
	Date               int64    `expr:"date"`
	Languages          []string `expr:"languages"`
	ContentID          string   `expr:"contentId"`
	ContentSource      string   `expr:"contentSource"`
}

func NewClassificationFromResult(r ClassificationResult) Classification {
	contentID := ""
	contentSource := ""

	if r.Content != nil {
		contentID = r.Content.ID
		contentSource = r.Content.Source
	}

	var dateVal int64
	if !r.Date.IsNil() {
		dateVal = r.Date.Time().Unix()
	}

	return Classification{
		ContentType:        ContentTypeToInt(r.ContentType),
		HasAttachedContent: r.Content != nil,
		HasBaseTitle:       r.BaseTitle.Valid,
		Date:               dateVal,
		Languages:          utils.Map(r.Languages.Slice(), func(l model.Language) string { return l.ID() }),
		ContentID:          contentID,
		ContentSource:      contentSource,
	}
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
