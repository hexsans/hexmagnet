package model

import (
	"github.com/hexsans/hexmagnet/internal/database/fts"
)

type ContentRef struct {
	Type   ContentType
	Source string
	ID     string
}

func (c Content) Ref() ContentRef {
	return ContentRef{
		Type:   c.Type,
		Source: c.Source,
		ID:     c.ID,
	}
}

func (c Content) Identifier(source string) (string, bool) {
	if c.Source == source {
		return c.ID, true
	}

	return "", false
}

type ExternalLink struct {
	Source string
	ID     string
	URL    string
}

func (c Content) ExternalLinks() []ExternalLink {
	links := make([]ExternalLink, 0)
	if link := getExternalLinkURL(c.Type, c.Source, c.ID); link.Valid {
		links = append(links, ExternalLink{
			Source: c.Source,
			URL:    link.String,
		})
	}

	return links
}

func getExternalLinkURL(contentType ContentType, source, id string) NullString {
	switch source {
	case SourceImdb:
		return NewNullString("https://www.imdb.com/title/" + id)
	case SourceTmdb:
		switch contentType {
		case ContentTypeTvShow:
			return NewNullString("https://www.themoviedb.org/tv/" + id)
		default:
			return NewNullString("https://www.themoviedb.org/movie/" + id)
		}
	case SourceTvdb:
		return NewNullString("https://www.thetvdb.com/dereferrer/series/" + id)
	}

	return NullString{}
}

func (c *Content) UpdateTsv() {
	tsv := fts.Tsvector{}
	tsv.AddText(c.Title, fts.TsvectorWeightA)

	if !c.ReleaseDate.IsNil() {
		tsv.AddText(c.ReleaseDate.YearString(), fts.TsvectorWeightB)
	}

	c.Tsv = tsv
}
