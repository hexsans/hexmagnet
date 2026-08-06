package tmdb

import (
	"context"
	"net/http"

	"github.com/go-resty/resty/v2"
)

type mockRequester struct {
	fn func(ctx context.Context, path string, queryParams map[string]string, result any) (*resty.Response, error)
}

func (m *mockRequester) Request(ctx context.Context, path string, queryParams map[string]string, result any) (*resty.Response, error) {
	return m.fn(ctx, path, queryParams, result)
}

type mockClient struct {
	validateErr error
}

func (m *mockClient) ValidateAccessToken(_ context.Context) error {
	return m.validateErr
}

func (mockClient) SearchMovie(_ context.Context, _ SearchMovieRequest) (SearchMovieResponse, error) {
	return SearchMovieResponse{}, nil
}

func (mockClient) MovieDetails(_ context.Context, _ MovieDetailsRequest) (MovieDetailsResponse, error) {
	return MovieDetailsResponse{}, nil
}

func (mockClient) SearchTv(_ context.Context, _ SearchTvRequest) (SearchTvResponse, error) {
	return SearchTvResponse{}, nil
}

func (mockClient) TvDetails(_ context.Context, _ TvDetailsRequest) (TvDetailsResponse, error) {
	return TvDetailsResponse{}, nil
}

func newRestyResponse(statusCode int) *resty.Response {
	return &resty.Response{
		RawResponse: &http.Response{
			StatusCode: statusCode,
			Status:     http.StatusText(statusCode),
		},
		Request: &resty.Request{},
	}
}
