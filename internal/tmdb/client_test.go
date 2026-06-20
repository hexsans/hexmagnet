package tmdb

import (
	"context"
	"errors"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/hexsans/hexmagnet/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_ValidateAccessToken_Success(t *testing.T) {
	t.Parallel()

	c := client{
		requester: &mockRequester{
			fn: func(_ context.Context, path string, _ map[string]string, _ any) (*resty.Response, error) {
				assert.Equal(t, "/authentication", path)
				return &resty.Response{}, nil
			},
		},
	}
	err := c.ValidateAccessToken(context.Background())
	assert.NoError(t, err)
}

func TestClient_ValidateAccessToken_Unauthorized(t *testing.T) {
	t.Parallel()

	c := client{
		requester: &mockRequester{
			fn: func(_ context.Context, _ string, _ map[string]string, _ any) (*resty.Response, error) {
				return nil, ErrUnauthorized
			},
		},
	}
	err := c.ValidateAccessToken(context.Background())
	assert.ErrorIs(t, err, ErrUnauthorized)
}

func TestClient_ValidateAccessToken_NotFound(t *testing.T) {
	t.Parallel()

	c := client{
		requester: &mockRequester{
			fn: func(_ context.Context, _ string, _ map[string]string, _ any) (*resty.Response, error) {
				return nil, ErrNotFound
			},
		},
	}
	err := c.ValidateAccessToken(context.Background())
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestClient_ValidateAccessToken_GenericError(t *testing.T) {
	t.Parallel()

	genericErr := errors.New("network error")
	c := client{
		requester: &mockRequester{
			fn: func(_ context.Context, _ string, _ map[string]string, _ any) (*resty.Response, error) {
				return nil, genericErr
			},
		},
	}
	err := c.ValidateAccessToken(context.Background())
	assert.ErrorIs(t, err, genericErr)
}

func TestClient_SearchMovie_Basic(t *testing.T) {
	t.Parallel()

	c := client{
		requester: &mockRequester{
			fn: func(_ context.Context, path string, queryParams map[string]string, _ any) (*resty.Response, error) {
				assert.Equal(t, "/search/movie", path)
				assert.Equal(t, "test", queryParams["query"])
				assert.Len(t, queryParams, 1)

				return &resty.Response{}, nil
			},
		},
	}
	_, err := c.SearchMovie(context.Background(), SearchMovieRequest{Query: "test"})
	assert.NoError(t, err)
}

func TestClient_SearchMovie_WithIncludeAdult(t *testing.T) {
	t.Parallel()

	c := client{
		requester: &mockRequester{
			fn: func(_ context.Context, _ string, queryParams map[string]string, _ any) (*resty.Response, error) {
				assert.Equal(t, "true", queryParams["include_adult"])
				return &resty.Response{}, nil
			},
		},
	}
	_, err := c.SearchMovie(context.Background(), SearchMovieRequest{Query: "q", IncludeAdult: true})
	assert.NoError(t, err)
}

func TestClient_SearchMovie_WithLanguage(t *testing.T) {
	t.Parallel()

	c := client{
		requester: &mockRequester{
			fn: func(_ context.Context, _ string, queryParams map[string]string, _ any) (*resty.Response, error) {
				assert.Equal(t, "en", queryParams["language"])
				return &resty.Response{}, nil
			},
		},
	}
	_, err := c.SearchMovie(context.Background(), SearchMovieRequest{Query: "q", Language: model.NewNullString("en")})
	assert.NoError(t, err)
}

func TestClient_SearchMovie_WithPrimaryReleaseYear(t *testing.T) {
	t.Parallel()

	c := client{
		requester: &mockRequester{
			fn: func(_ context.Context, _ string, queryParams map[string]string, _ any) (*resty.Response, error) {
				assert.Equal(t, "2024", queryParams["primary_release_year"])
				return &resty.Response{}, nil
			},
		},
	}
	_, err := c.SearchMovie(context.Background(), SearchMovieRequest{Query: "q", PrimaryReleaseYear: model.Year(2024)})
	assert.NoError(t, err)
}

func TestClient_SearchMovie_WithYear(t *testing.T) {
	t.Parallel()

	c := client{
		requester: &mockRequester{
			fn: func(_ context.Context, _ string, queryParams map[string]string, _ any) (*resty.Response, error) {
				assert.Equal(t, "2023", queryParams["year"])
				return &resty.Response{}, nil
			},
		},
	}
	_, err := c.SearchMovie(context.Background(), SearchMovieRequest{Query: "q", Year: model.Year(2023)})
	assert.NoError(t, err)
}

func TestClient_SearchMovie_WithRegion(t *testing.T) {
	t.Parallel()

	c := client{
		requester: &mockRequester{
			fn: func(_ context.Context, _ string, queryParams map[string]string, _ any) (*resty.Response, error) {
				assert.Equal(t, "US", queryParams["region"])
				return &resty.Response{}, nil
			},
		},
	}
	_, err := c.SearchMovie(context.Background(), SearchMovieRequest{Query: "q", Region: model.NewNullString("US")})
	assert.NoError(t, err)
}

func TestClient_SearchMovie_AllOptions(t *testing.T) {
	t.Parallel()

	c := client{
		requester: &mockRequester{
			fn: func(_ context.Context, path string, queryParams map[string]string, _ any) (*resty.Response, error) {
				assert.Equal(t, "/search/movie", path)
				assert.Equal(t, "test", queryParams["query"])
				assert.Equal(t, "true", queryParams["include_adult"])
				assert.Equal(t, "en", queryParams["language"])
				assert.Equal(t, "2024", queryParams["primary_release_year"])
				assert.Equal(t, "2023", queryParams["year"])
				assert.Equal(t, "US", queryParams["region"])

				return &resty.Response{}, nil
			},
		},
	}
	_, err := c.SearchMovie(context.Background(), SearchMovieRequest{
		Query:              "test",
		IncludeAdult:       true,
		Language:           model.NewNullString("en"),
		PrimaryReleaseYear: model.Year(2024),
		Year:               model.Year(2023),
		Region:             model.NewNullString("US"),
	})
	assert.NoError(t, err)
}

func TestClient_SearchMovie_Error(t *testing.T) {
	t.Parallel()

	c := client{
		requester: &mockRequester{
			fn: func(_ context.Context, _ string, _ map[string]string, _ any) (*resty.Response, error) {
				return nil, errors.New("search error")
			},
		},
	}
	resp, err := c.SearchMovie(context.Background(), SearchMovieRequest{Query: "test"})
	require.Error(t, err)
	assert.Empty(t, resp)
}

func TestClient_MovieDetails_Basic(t *testing.T) {
	t.Parallel()

	c := client{
		requester: &mockRequester{
			fn: func(_ context.Context, path string, queryParams map[string]string, _ any) (*resty.Response, error) {
				assert.Equal(t, "/movie/123", path)
				assert.Empty(t, queryParams)

				return &resty.Response{}, nil
			},
		},
	}
	_, err := c.MovieDetails(context.Background(), MovieDetailsRequest{ID: 123})
	assert.NoError(t, err)
}

func TestClient_MovieDetails_WithAppendToResponse(t *testing.T) {
	t.Parallel()

	c := client{
		requester: &mockRequester{
			fn: func(_ context.Context, path string, queryParams map[string]string, _ any) (*resty.Response, error) {
				assert.Equal(t, "/movie/456", path)
				assert.Equal(t, "credits,similar", queryParams["append_to_response"])

				return &resty.Response{}, nil
			},
		},
	}
	_, err := c.MovieDetails(context.Background(), MovieDetailsRequest{
		ID:               456,
		AppendToResponse: []string{"credits", "similar"},
	})
	assert.NoError(t, err)
}

func TestClient_MovieDetails_WithLanguage(t *testing.T) {
	t.Parallel()

	c := client{
		requester: &mockRequester{
			fn: func(_ context.Context, path string, queryParams map[string]string, _ any) (*resty.Response, error) {
				assert.Equal(t, "/movie/789", path)
				assert.Equal(t, "fr", queryParams["language"])

				return &resty.Response{}, nil
			},
		},
	}
	_, err := c.MovieDetails(context.Background(), MovieDetailsRequest{
		ID:       789,
		Language: model.NewNullString("fr"),
	})
	assert.NoError(t, err)
}

func TestClient_MovieDetails_NotFound(t *testing.T) {
	t.Parallel()

	c := client{
		requester: &mockRequester{
			fn: func(_ context.Context, _ string, _ map[string]string, _ any) (*resty.Response, error) {
				return nil, ErrNotFound
			},
		},
	}
	resp, err := c.MovieDetails(context.Background(), MovieDetailsRequest{ID: 999})
	require.ErrorIs(t, err, ErrNotFound)
	assert.Empty(t, resp)
}

func TestClient_SearchTv_Basic(t *testing.T) {
	t.Parallel()

	c := client{
		requester: &mockRequester{
			fn: func(_ context.Context, path string, queryParams map[string]string, _ any) (*resty.Response, error) {
				assert.Equal(t, "/search/tv", path)
				assert.Equal(t, "show", queryParams["query"])
				assert.Len(t, queryParams, 1)

				return &resty.Response{}, nil
			},
		},
	}
	_, err := c.SearchTv(context.Background(), SearchTvRequest{Query: "show"})
	assert.NoError(t, err)
}

func TestClient_SearchTv_WithFirstAirDateYear(t *testing.T) {
	t.Parallel()

	c := client{
		requester: &mockRequester{
			fn: func(_ context.Context, _ string, queryParams map[string]string, _ any) (*resty.Response, error) {
				assert.Equal(t, "2022", queryParams["first_air_date_year"])
				return &resty.Response{}, nil
			},
		},
	}
	_, err := c.SearchTv(context.Background(), SearchTvRequest{Query: "q", FirstAirDateYear: model.Year(2022)})
	assert.NoError(t, err)
}

func TestClient_SearchTv_WithIncludeAdult(t *testing.T) {
	t.Parallel()

	c := client{
		requester: &mockRequester{
			fn: func(_ context.Context, _ string, queryParams map[string]string, _ any) (*resty.Response, error) {
				assert.Equal(t, "true", queryParams["include_adult"])
				return &resty.Response{}, nil
			},
		},
	}
	_, err := c.SearchTv(context.Background(), SearchTvRequest{Query: "q", IncludeAdult: true})
	assert.NoError(t, err)
}

func TestClient_SearchTv_WithLanguage(t *testing.T) {
	t.Parallel()

	c := client{
		requester: &mockRequester{
			fn: func(_ context.Context, _ string, queryParams map[string]string, _ any) (*resty.Response, error) {
				assert.Equal(t, "de", queryParams["language"])
				return &resty.Response{}, nil
			},
		},
	}
	_, err := c.SearchTv(context.Background(), SearchTvRequest{Query: "q", Language: model.NewNullString("de")})
	assert.NoError(t, err)
}

func TestClient_SearchTv_AllOptions(t *testing.T) {
	t.Parallel()

	c := client{
		requester: &mockRequester{
			fn: func(_ context.Context, path string, queryParams map[string]string, _ any) (*resty.Response, error) {
				assert.Equal(t, "/search/tv", path)
				assert.Equal(t, "show", queryParams["query"])
				assert.Equal(t, "true", queryParams["include_adult"])
				assert.Equal(t, "en", queryParams["language"])
				assert.Equal(t, "2023", queryParams["first_air_date_year"])

				return &resty.Response{}, nil
			},
		},
	}
	_, err := c.SearchTv(context.Background(), SearchTvRequest{
		Query:            "show",
		IncludeAdult:     true,
		Language:         model.NewNullString("en"),
		FirstAirDateYear: model.Year(2023),
	})
	assert.NoError(t, err)
}

func TestClient_SearchTv_Error(t *testing.T) {
	t.Parallel()

	c := client{
		requester: &mockRequester{
			fn: func(_ context.Context, _ string, _ map[string]string, _ any) (*resty.Response, error) {
				return nil, errors.New("tv search error")
			},
		},
	}
	resp, err := c.SearchTv(context.Background(), SearchTvRequest{Query: "q"})
	require.Error(t, err)
	assert.Empty(t, resp)
}

func TestClient_TvDetails_Basic(t *testing.T) {
	t.Parallel()

	c := client{
		requester: &mockRequester{
			fn: func(_ context.Context, path string, queryParams map[string]string, _ any) (*resty.Response, error) {
				assert.Equal(t, "/tv/42", path)
				assert.Empty(t, queryParams)

				return &resty.Response{}, nil
			},
		},
	}
	_, err := c.TvDetails(context.Background(), TvDetailsRequest{SeriesID: 42})
	assert.NoError(t, err)
}

func TestClient_TvDetails_WithAppendToResponse(t *testing.T) {
	t.Parallel()

	c := client{
		requester: &mockRequester{
			fn: func(_ context.Context, path string, queryParams map[string]string, _ any) (*resty.Response, error) {
				assert.Equal(t, "/tv/100", path)
				assert.Equal(t, "videos,credits", queryParams["append_to_response"])

				return &resty.Response{}, nil
			},
		},
	}
	_, err := c.TvDetails(context.Background(), TvDetailsRequest{
		SeriesID:         100,
		AppendToResponse: []string{"videos", "credits"},
	})
	assert.NoError(t, err)
}

func TestClient_TvDetails_WithLanguage(t *testing.T) {
	t.Parallel()

	c := client{
		requester: &mockRequester{
			fn: func(_ context.Context, path string, queryParams map[string]string, _ any) (*resty.Response, error) {
				assert.Equal(t, "/tv/200", path)
				assert.Equal(t, "ja", queryParams["language"])

				return &resty.Response{}, nil
			},
		},
	}
	_, err := c.TvDetails(context.Background(), TvDetailsRequest{
		SeriesID: 200,
		Language: model.NewNullString("ja"),
	})
	assert.NoError(t, err)
}

func TestClient_TvDetails_NotFound(t *testing.T) {
	t.Parallel()

	c := client{
		requester: &mockRequester{
			fn: func(_ context.Context, _ string, _ map[string]string, _ any) (*resty.Response, error) {
				return nil, ErrNotFound
			},
		},
	}
	resp, err := c.TvDetails(context.Background(), TvDetailsRequest{SeriesID: 999})
	require.ErrorIs(t, err, ErrNotFound)
	assert.Empty(t, resp)
}
