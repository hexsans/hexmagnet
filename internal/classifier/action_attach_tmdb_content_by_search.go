package classifier

import (
	"github.com/hexsans/hexmagnet/internal/model"
)

const attachTmdbContentBySearchName = "attach_tmdb_content_by_search"

type attachTmdbContentBySearchAction struct{}

func (attachTmdbContentBySearchAction) name() string {
	return attachTmdbContentBySearchName
}

var attachTmdbContentBySearchPayloadSpec = payloadLiteral[string]{
	literal:     attachTmdbContentBySearchName,
	description: "Attempt to attach content from the TMDB API with a search on the torrent name",
}

func (attachTmdbContentBySearchAction) compileAction(ctx compilerContext) (action, error) {
	if _, err := attachTmdbContentBySearchPayloadSpec.Unmarshal(ctx); err != nil {
		return action{}, ctx.error(err)
	}

	return action{
		run: func(ctx executionContext) (ClassificationResult, error) {
			cl := ctx.result
			if !cl.BaseTitle.Valid {
				return cl, ErrUnmatched
			}

			var content *model.Content

			switch cl.ContentType.ContentType {
			case model.ContentTypeTvShow:
				result, searchErr := ctx.tmdbSearchTVShow(cl.BaseTitle.String, cl.Date.Year)
				if searchErr != nil {
					return cl, searchErr
				}

				content = &result
			default:
				result, searchErr := ctx.tmdbSearchMovie(cl.BaseTitle.String, cl.Date.Year)
				if searchErr != nil {
					return cl, searchErr
				}

				content = &result
			}

			cl.AttachContent(content)

			return cl, nil
		},
	}, nil
}

func (attachTmdbContentBySearchAction) JSONSchema() JSONSchema {
	return attachTmdbContentBySearchPayloadSpec.JSONSchema()
}
