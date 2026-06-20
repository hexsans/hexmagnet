package tmdb

import (
	"strconv"

	"github.com/hexsans/hexmagnet/internal/model"
)

func MovieDetailsToMovieModel(details MovieDetailsResponse) (movie model.Content, err error) {
	releaseDate := model.Date{}

	if details.ReleaseDate != "" {
		parsedDate, parseDateErr := model.NewDateFromIsoString(details.ReleaseDate)
		if parseDateErr != nil {
			err = parseDateErr
			return
		}

		releaseDate = parsedDate
	}

	contentType := model.ContentTypeMovie

	if details.Adult {
		contentType = model.ContentTypeOther
	}

	return model.Content{
		Type:        contentType,
		Source:      model.SourceTmdb,
		ID:          strconv.Itoa(int(details.ID)),
		Title:       details.Title,
		ReleaseDate: releaseDate,
		Adult:       model.NewNullBool(details.Adult),
		Overview: model.NullString{
			String: details.Overview,
			Valid:  details.Overview != "",
		},
		Popularity:  model.NewNullFloat32(details.Popularity),
		VoteAverage: model.NewNullFloat32(details.VoteAverage),
		VoteCount:   model.NewNullUint(uint(details.VoteCount)),
	}, nil
}

func TvShowDetailsToTvShowModel(details TvDetailsResponse) (movie model.Content, err error) {
	firstAirDate := model.Date{}

	if details.FirstAirDate != "" {
		parsedDate, parseDateErr := model.NewDateFromIsoString(details.FirstAirDate)
		if parseDateErr != nil {
			err = parseDateErr
			return
		}

		firstAirDate = parsedDate
	}

	return model.Content{
		Type:        model.ContentTypeTvShow,
		Source:      model.SourceTmdb,
		ID:          strconv.Itoa(int(details.ID)),
		Title:       details.Name,
		ReleaseDate: firstAirDate,
		Overview: model.NullString{
			String: details.Overview,
			Valid:  details.Overview != "",
		},
		Popularity:  model.NewNullFloat32(details.Popularity),
		VoteAverage: model.NewNullFloat32(details.VoteAverage),
		VoteCount:   model.NewNullUint(uint(details.VoteCount)),
	}, nil
}
