package tmdb

import (
	"testing"

	"github.com/hexsans/hexmagnet/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMovieDetailsToMovieModel_Full(t *testing.T) {
	t.Parallel()

	details := MovieDetailsResponse{
		ID:          42,
		Title:       "Test Movie",
		Overview:    "A test movie overview",
		ReleaseDate: "2024-03-15",
		Adult:       false,
		Popularity:  10.5,
		VoteAverage: 7.5,
		VoteCount:   1000,
	}
	content, err := MovieDetailsToMovieModel(details)
	require.NoError(t, err)
	assert.Equal(t, model.ContentTypeMovie, content.Type)
	assert.Equal(t, model.SourceTmdb, content.Source)
	assert.Equal(t, "42", content.ID)
	assert.Equal(t, "Test Movie", content.Title)
	assert.True(t, content.Overview.Valid)
	assert.Equal(t, "A test movie overview", content.Overview.String)
	assert.False(t, content.Adult.Bool)
	assert.True(t, content.Adult.Valid)
	assert.InDelta(t, 10.5, content.Popularity.Float32, 0.001)
	assert.InDelta(t, 7.5, content.VoteAverage.Float32, 0.001)
	assert.Equal(t, uint(1000), content.VoteCount.Uint)

	expectedDate, err := model.NewDateFromIsoString("2024-03-15")
	require.NoError(t, err)
	assert.Equal(t, expectedDate, content.ReleaseDate)
}

func TestMovieDetailsToMovieModel_Adult(t *testing.T) {
	t.Parallel()

	details := MovieDetailsResponse{
		ID:          1,
		Title:       "Adult Movie",
		ReleaseDate: "2024-01-01",
		Adult:       true,
	}
	content, err := MovieDetailsToMovieModel(details)
	require.NoError(t, err)
	assert.Equal(t, model.ContentTypeOther, content.Type)
	assert.True(t, content.Adult.Bool)
	assert.True(t, content.Adult.Valid)
}

func TestMovieDetailsToMovieModel_EmptyReleaseDate(t *testing.T) {
	t.Parallel()

	details := MovieDetailsResponse{
		ID:    2,
		Title: "No Release Date",
	}
	content, err := MovieDetailsToMovieModel(details)
	require.NoError(t, err)
	assert.True(t, content.ReleaseDate.IsNil())
}

func TestMovieDetailsToMovieModel_InvalidReleaseDate(t *testing.T) {
	t.Parallel()

	details := MovieDetailsResponse{
		ID:          3,
		Title:       "Bad Date",
		ReleaseDate: "not-a-date",
	}
	_, err := MovieDetailsToMovieModel(details)
	assert.Error(t, err)
}

func TestMovieDetailsToMovieModel_EmptyOverview(t *testing.T) {
	t.Parallel()

	details := MovieDetailsResponse{
		ID:          4,
		Title:       "No Overview",
		ReleaseDate: "2024-01-01",
	}
	content, err := MovieDetailsToMovieModel(details)
	require.NoError(t, err)
	assert.False(t, content.Overview.Valid)
	assert.Empty(t, content.Overview.String)
}

func TestMovieDetailsToMovieModel_ZeroPopularity(t *testing.T) {
	t.Parallel()

	details := MovieDetailsResponse{
		ID:          5,
		Title:       "Zero Pop",
		ReleaseDate: "2024-01-01",
		Popularity:  0,
	}
	content, err := MovieDetailsToMovieModel(details)
	require.NoError(t, err)
	assert.InDelta(t, 0.0, content.Popularity.Float32, 0.001)
	assert.True(t, content.Popularity.Valid)
}

func TestMovieDetailsToMovieModel_ZeroVoteCount(t *testing.T) {
	t.Parallel()

	details := MovieDetailsResponse{
		ID:          6,
		Title:       "No Votes",
		ReleaseDate: "2024-01-01",
		VoteCount:   0,
	}
	content, err := MovieDetailsToMovieModel(details)
	require.NoError(t, err)
	assert.Equal(t, uint(0), content.VoteCount.Uint)
	assert.True(t, content.VoteCount.Valid)
}

func TestTvShowDetailsToTvShowModel_Full(t *testing.T) {
	t.Parallel()

	details := TvDetailsResponse{
		ID:           99,
		Name:         "Test Show",
		Overview:     "A TV show overview",
		FirstAirDate: "2023-06-01",
		Popularity:   20.0,
		VoteAverage:  8.0,
		VoteCount:    500,
	}
	content, err := TvShowDetailsToTvShowModel(details)
	require.NoError(t, err)
	assert.Equal(t, model.ContentTypeTvShow, content.Type)
	assert.Equal(t, model.SourceTmdb, content.Source)
	assert.Equal(t, "99", content.ID)
	assert.Equal(t, "Test Show", content.Title)
	assert.True(t, content.Overview.Valid)
	assert.Equal(t, "A TV show overview", content.Overview.String)
	assert.InDelta(t, 20.0, content.Popularity.Float32, 0.001)
	assert.InDelta(t, 8.0, content.VoteAverage.Float32, 0.001)
	assert.Equal(t, uint(500), content.VoteCount.Uint)

	expectedDate, err := model.NewDateFromIsoString("2023-06-01")
	require.NoError(t, err)
	assert.Equal(t, expectedDate, content.ReleaseDate)
}

func TestTvShowDetailsToTvShowModel_EmptyFirstAirDate(t *testing.T) {
	t.Parallel()

	details := TvDetailsResponse{
		ID:   100,
		Name: "No Air Date",
	}
	content, err := TvShowDetailsToTvShowModel(details)
	require.NoError(t, err)
	assert.True(t, content.ReleaseDate.IsNil())
}

func TestTvShowDetailsToTvShowModel_InvalidFirstAirDate(t *testing.T) {
	t.Parallel()

	details := TvDetailsResponse{
		ID:           101,
		Name:         "Bad Date",
		FirstAirDate: "invalid-date",
	}
	_, err := TvShowDetailsToTvShowModel(details)
	assert.Error(t, err)
}

func TestTvShowDetailsToTvShowModel_EmptyOverview(t *testing.T) {
	t.Parallel()

	details := TvDetailsResponse{
		ID:           102,
		Name:         "No Overview",
		FirstAirDate: "2024-01-01",
	}
	content, err := TvShowDetailsToTvShowModel(details)
	require.NoError(t, err)
	assert.False(t, content.Overview.Valid)
	assert.Empty(t, content.Overview.String)
}

func TestTvShowDetailsToTvShowModel_ZeroPopularity(t *testing.T) {
	t.Parallel()

	details := TvDetailsResponse{
		ID:           103,
		Name:         "Zero Pop",
		FirstAirDate: "2024-01-01",
		Popularity:   0,
	}
	content, err := TvShowDetailsToTvShowModel(details)
	require.NoError(t, err)
	assert.InDelta(t, 0.0, content.Popularity.Float32, 0.001)
	assert.True(t, content.Popularity.Valid)
}
