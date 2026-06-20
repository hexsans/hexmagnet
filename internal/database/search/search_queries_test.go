package dbsearch

import (
	"testing"

	search "github.com/hexsans/hexmagnet/internal/search"
	"github.com/stretchr/testify/require"
)

func countPlaceholders(query string) int {
	if query == "" {
		return 0
	}

	for i := 1; ; i++ {
		found := false

		for j := range len(query) - 1 {
			if query[j] == '$' {
				num := 0

				k := j + 1
				for k < len(query) && query[k] >= '0' && query[k] <= '9' {
					num = num*10 + int(query[k]-'0')
					k++
				}

				if num == i {
					found = true
					break
				}
			}
		}

		if !found {
			return i - 1
		}
	}
}

func TestBuildTorrentContentQueries_NoQueryDefaultSort(t *testing.T) {
	t.Parallel()

	q := buildTorrentContentQueries(search.TorrentSearchParams{
		TotalCount:  true,
		HasNextPage: true,
		Limit:       20,
	}, "WHERE 1=1", []any{}, 1, 0)

	// Main query + count + hasNext = 3 placeholders (where, order-by must match)
	require.Len(t, q.args, countPlaceholders(q.query))
	require.Len(t, q.countArgs, countPlaceholders(q.countQuery))
	require.Len(t, q.hasNextArgs, countPlaceholders(q.hasNextQuery))
}

func TestBuildTorrentContentQueries_WithQueryRelevanceSort(t *testing.T) {
	t.Parallel()

	q := buildTorrentContentQueries(search.TorrentSearchParams{
		QueryString: "test query",
		TotalCount:  true,
		HasNextPage: true,
		Limit:       20,
		OrderBy: []search.TorrentSearchOrder{
			{Field: search.FieldRelevance, Direction: search.SortDesc},
		},
	}, "WHERE tc.tsv @@ $1::tsquery", []any{"test"}, 2, 0)

	// Where clause has $1 (tsquery), relevance sort adds $2 (tsquery), then limit/offset are $3/$4
	// Main query args: [tsquery_for_where, tsquery_for_order_by, limit, offset] = 4
	require.Len(t, q.args, countPlaceholders(q.query),
		"main query args must match placeholders")
	require.Len(t, q.args, 4)

	// Count query only uses whereClause which has just $1
	// countArgs should be args[:preOrderByArgsLen] = [tsquery_for_where] = 1
	require.Len(t, q.countArgs, countPlaceholders(q.countQuery),
		"count query args must match placeholders (bug: order-by args leaking)")
	require.Len(t, q.countArgs, 1,
		"count query must not include order-by args")

	// Has next query uses whereClause ($1) + orderByClause ($2) + offset ($3)
	require.Len(t, q.hasNextArgs, countPlaceholders(q.hasNextQuery),
		"hasNext query args must match placeholders")
}

func TestBuildTorrentContentQueries_WithQueryNoRelevanceSort(t *testing.T) {
	t.Parallel()

	q := buildTorrentContentQueries(search.TorrentSearchParams{
		QueryString: "test query",
		TotalCount:  true,
		HasNextPage: true,
		Limit:       20,
		OrderBy: []search.TorrentSearchOrder{
			{Field: search.FieldSeeders, Direction: search.SortDesc},
		},
	}, "WHERE tc.tsv @@ $1::tsquery", []any{"test"}, 2, 0)

	// Only where clause has $1, no order-by adds args, limit/offset are $2/$3
	require.Len(t, q.args, countPlaceholders(q.query),
		"main query args must match placeholders")
	require.Len(t, q.args, 3)

	// Count query: only $1 in where clause, no order-by args
	require.Len(t, q.countArgs, countPlaceholders(q.countQuery),
		"count query args must match placeholders")
	require.Len(t, q.countArgs, 1)
}

func TestBuildTorrentContentQueries_MultipleFiltersRelevanceSort(t *testing.T) {
	t.Parallel()

	q := buildTorrentContentQueries(search.TorrentSearchParams{
		QueryString:  "test",
		TotalCount:   true,
		HasNextPage:  true,
		ContentTypes: []string{"movie", "tv_show"},
		OrderBy: []search.TorrentSearchOrder{
			{Field: search.FieldRelevance, Direction: search.SortDesc},
		},
	}, "WHERE tc.tsv @@ $1::tsquery AND tc.content_type IN ($2, $3)",
		[]any{"tsquery", "movie", "tv_show"}, 4, 0)

	// Where: $1 (tsquery), $2 (movie), $3 (tv_show)
	// Order by relevance: $4 (tsquery for ts_rank)
	// Limit: $5, Offset: $6
	require.Len(t, q.args, countPlaceholders(q.query))
	require.Len(t, q.args, 6)

	// Count: only where clause ($1, $2, $3) = 3 args
	require.Len(t, q.countArgs, countPlaceholders(q.countQuery))
	require.Len(t, q.countArgs, 3)

	// Has next: where ($1-$3) + order by ($4) + offset ($5)
	require.Len(t, q.hasNextArgs, countPlaceholders(q.hasNextQuery))
}

func TestBuildTorrentContentQueries_NoTotalCount(t *testing.T) {
	t.Parallel()

	q := buildTorrentContentQueries(search.TorrentSearchParams{
		QueryString: "test",
		HasNextPage: true,
		OrderBy: []search.TorrentSearchOrder{
			{Field: search.FieldRelevance, Direction: search.SortDesc},
		},
	}, "WHERE tc.tsv @@ $1::tsquery", []any{"test"}, 2, 0)

	// countQuery/countArgs should be empty when TotalCount is false
	require.Empty(t, q.countQuery)
	require.Nil(t, q.countArgs)
}

func TestBuildTorrentContentQueries_NoHasNextPage(t *testing.T) {
	t.Parallel()

	q := buildTorrentContentQueries(search.TorrentSearchParams{
		QueryString: "test",
		TotalCount:  true,
		OrderBy: []search.TorrentSearchOrder{
			{Field: search.FieldRelevance, Direction: search.SortDesc},
		},
	}, "WHERE tc.tsv @@ $1::tsquery", []any{"test"}, 2, 0)

	require.Empty(t, q.hasNextQuery)
	require.Nil(t, q.hasNextArgs)
}

func TestBuildTorrentContentQueries_EmptyWhere(t *testing.T) {
	t.Parallel()

	q := buildTorrentContentQueries(search.TorrentSearchParams{
		TotalCount:  true,
		HasNextPage: true,
		OrderBy: []search.TorrentSearchOrder{
			{Field: search.FieldSeeders, Direction: search.SortDesc},
		},
	}, "", []any{}, 1, 0)

	// No where clause, no order-by args, limit/offset are $1/$2
	require.Len(t, q.args, countPlaceholders(q.query))
	require.Len(t, q.args, 2)

	require.Len(t, q.countArgs, countPlaceholders(q.countQuery))
	require.Empty(t, q.countArgs)
}

func TestBuildTorrentContentQueries_HasNextArgsDoesNotCorruptMainArgs(t *testing.T) {
	t.Parallel()

	// Regression test: hasNextArgs append was corrupting the main query's
	// LIMIT arg due to shared backing array. The 3-index slice expression
	// fix isolates the capacity so the main args remain untouched.
	q := buildTorrentContentQueries(search.TorrentSearchParams{
		Limit:       15,
		Offset:      30,
		HasNextPage: true,
	}, "", []any{}, 1, 30)

	// Main query args must have INTACT limit value
	require.Len(t, q.args, 2)
	require.Equal(t, int32(15), q.args[0],
		"hasNextArgs must not corrupt main query limit arg")
	require.Equal(t, int32(30), q.args[1],
		"hasNextArgs must not corrupt main query offset arg")

	// hasNext args should have the correct value (offset+limit)
	require.Len(t, q.hasNextArgs, 1)
	require.Equal(t, int32(45), q.hasNextArgs[0])
}

func TestBuildTorrentContentQueries_HasNextArgsDoesNotCorruptMainArgs_WithWhere(t *testing.T) {
	t.Parallel()

	// Same regression test but with WHERE clause args present.
	// When QueryString is set with no OrderBy, default relevance sort
	// adds one tsquery arg (repeated), so args = [whereTsquery, orderTsquery, limit, offset].
	q := buildTorrentContentQueries(search.TorrentSearchParams{
		QueryString: "test",
		Limit:       15,
		Offset:      30,
		HasNextPage: true,
	}, "WHERE tc.tsv @@ $1::tsquery", []any{"tsquery"}, 2, 30)

	require.Len(t, q.args, 4)
	require.Equal(t, int32(15), q.args[2],
		"hasNextArgs must not corrupt main query limit arg")
	require.Equal(t, int32(30), q.args[3],
		"hasNextArgs must not corrupt main query offset arg")

	require.Len(t, q.hasNextArgs, 3)
	require.Equal(t, int32(45), q.hasNextArgs[2])
}
