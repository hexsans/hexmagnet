package dbsearch

import (
	"testing"

	search "github.com/hexsans/hexmagnet/internal/search"
	"github.com/stretchr/testify/require"
)

func TestBuildTorrentContentOrderBy_DefaultNoQuery(t *testing.T) {
	t.Parallel()

	result := buildTorrentContentOrderBy(search.TorrentSearchParams{}, &[]any{}, new(int))
	require.Equal(t, "ORDER BY t.created_at DESC, t.info_hash DESC", result)
}

func TestBuildTorrentContentOrderBy_DefaultWithQuery(t *testing.T) {
	t.Parallel()

	args := []any{}
	argIdx := new(int)
	*argIdx = 1
	result := buildTorrentContentOrderBy(search.TorrentSearchParams{
		QueryString: "test query",
	}, &args, argIdx)
	require.Equal(t, "ORDER BY ts_rank(t.tsv, $1::tsquery) DESC, t.info_hash DESC", result)
	require.Len(t, args, 1)
}

func TestBuildTorrentContentOrderBy_SingleFieldDesc(t *testing.T) {
	t.Parallel()

	result := buildTorrentContentOrderBy(search.TorrentSearchParams{
		OrderBy: []search.TorrentSearchOrder{
			{Field: search.FieldSeeders, Direction: search.SortDesc},
		},
	}, &[]any{}, new(int))
	require.Equal(t, "ORDER BY COALESCE(t.seeders, -1) DESC, t.info_hash DESC", result)
}

func TestBuildTorrentContentOrderBy_SingleFieldAsc(t *testing.T) {
	t.Parallel()

	result := buildTorrentContentOrderBy(search.TorrentSearchParams{
		OrderBy: []search.TorrentSearchOrder{
			{Field: search.FieldSize, Direction: search.SortAsc},
		},
	}, &[]any{}, new(int))
	require.Equal(t, "ORDER BY t.size ASC, t.info_hash ASC", result)
}

func TestBuildTorrentContentOrderBy_MultipleFields(t *testing.T) {
	t.Parallel()

	result := buildTorrentContentOrderBy(search.TorrentSearchParams{
		OrderBy: []search.TorrentSearchOrder{
			{Field: search.FieldSeeders, Direction: search.SortDesc},
			{Field: search.FieldSize, Direction: search.SortDesc},
		},
	}, &[]any{}, new(int))
	require.Equal(t, "ORDER BY COALESCE(t.seeders, -1) DESC, t.size DESC, t.info_hash DESC", result)
}

func TestBuildTorrentContentOrderBy_MultipleFieldsMixedDir(t *testing.T) {
	t.Parallel()

	result := buildTorrentContentOrderBy(search.TorrentSearchParams{
		OrderBy: []search.TorrentSearchOrder{
			{Field: search.FieldCreatedAt, Direction: search.SortDesc},
			{Field: search.FieldName, Direction: search.SortAsc},
		},
	}, &[]any{}, new(int))
	require.Equal(t, "ORDER BY t.created_at DESC, t.name ASC, t.info_hash ASC", result)
}

func TestBuildTorrentContentOrderBy_ThreeFields(t *testing.T) {
	t.Parallel()

	result := buildTorrentContentOrderBy(search.TorrentSearchParams{
		OrderBy: []search.TorrentSearchOrder{
			{Field: search.FieldSeeders, Direction: search.SortDesc},
			{Field: search.FieldLeechers, Direction: search.SortDesc},
			{Field: search.FieldSize, Direction: search.SortAsc},
		},
	}, &[]any{}, new(int))
	require.Equal(t, "ORDER BY COALESCE(t.seeders, -1) DESC, COALESCE(t.leechers, -1) DESC, t.size ASC, t.info_hash ASC", result)
}

func TestBuildTorrentContentOrderBy_RelevanceWithQuery(t *testing.T) {
	t.Parallel()

	args := []any{}
	argIdx := new(int)
	*argIdx = 1
	result := buildTorrentContentOrderBy(search.TorrentSearchParams{
		QueryString: "test query",
		OrderBy: []search.TorrentSearchOrder{
			{Field: search.FieldRelevance, Direction: search.SortDesc},
		},
	}, &args, argIdx)
	require.Equal(t, "ORDER BY ts_rank(t.tsv, $1::tsquery) DESC, t.info_hash DESC", result)
	require.Len(t, args, 1)
}

func TestBuildTorrentContentOrderBy_RelevanceWithoutQuery(t *testing.T) {
	t.Parallel()

	result := buildTorrentContentOrderBy(search.TorrentSearchParams{
		OrderBy: []search.TorrentSearchOrder{
			{Field: search.FieldRelevance, Direction: search.SortDesc},
		},
	}, &[]any{}, new(int))
	require.Equal(t, "ORDER BY t.created_at DESC, t.info_hash DESC", result)
}

func TestBuildTorrentContentOrderBy_RelevanceSkippedFallbackToNext(t *testing.T) {
	t.Parallel()

	result := buildTorrentContentOrderBy(search.TorrentSearchParams{
		OrderBy: []search.TorrentSearchOrder{
			{Field: search.FieldRelevance, Direction: search.SortDesc},
			{Field: search.FieldSeeders, Direction: search.SortDesc},
		},
	}, &[]any{}, new(int))
	require.Equal(t, "ORDER BY t.created_at DESC, COALESCE(t.seeders, -1) DESC, t.info_hash DESC", result)
}

func TestBuildTorrentContentOrderBy_AllFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		field    search.TorrentSearchField
		expected string
	}{
		{"created_at", search.FieldCreatedAt, "ORDER BY t.created_at DESC, t.info_hash DESC"},
		{"updated_at", search.FieldUpdatedAt, "ORDER BY t.updated_at DESC, t.info_hash DESC"},
		{"size", search.FieldSize, "ORDER BY t.size DESC, t.info_hash DESC"},
		{"files_count", search.FieldFilesCount, "ORDER BY COALESCE(t.files_count, 0) DESC, t.info_hash DESC"},
		{"seeders", search.FieldSeeders, "ORDER BY COALESCE(t.seeders, -1) DESC, t.info_hash DESC"},
		{"leechers", search.FieldLeechers, "ORDER BY COALESCE(t.leechers, -1) DESC, t.info_hash DESC"},
		{"name", search.FieldName, "ORDER BY t.name DESC, t.info_hash DESC"},
		{"info_hash", search.FieldInfoHash, "ORDER BY t.info_hash DESC, t.info_hash DESC"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := buildTorrentContentOrderBy(search.TorrentSearchParams{
				OrderBy: []search.TorrentSearchOrder{
					{Field: tt.field, Direction: search.SortDesc},
				},
			}, &[]any{}, new(int))
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildTorrentContentOrderBy_AllFieldsMultiple(t *testing.T) {
	t.Parallel()

	result := buildTorrentContentOrderBy(search.TorrentSearchParams{
		OrderBy: []search.TorrentSearchOrder{
			{Field: search.FieldFilesCount, Direction: search.SortDesc},
			{Field: search.FieldLeechers, Direction: search.SortAsc},
			{Field: search.FieldUpdatedAt, Direction: search.SortDesc},
			{Field: search.FieldInfoHash, Direction: search.SortAsc},
		},
	}, &[]any{}, new(int))
	require.Equal(
		t,
		"ORDER BY COALESCE(t.files_count, 0) DESC, COALESCE(t.leechers, -1) ASC, t.updated_at DESC, t.info_hash ASC, t.info_hash ASC",
		result,
	)
}
