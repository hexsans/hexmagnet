package dbsearch

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/hexsans/hexmagnet/internal/database/fts"
	search "github.com/hexsans/hexmagnet/internal/search"
	"github.com/hexsans/hexmagnet/internal/utils"
)

func (s *pgSearch) TorrentSearch(ctx context.Context, params search.TorrentSearchParams) (search.TorrentSearchResult, error) {
	args := []any{}
	argIdx := 1

	if params.Barrier != "" {
		if decoded, err := search.DecodeBarrier(params.Barrier); err == nil {
			params.Barrier = decoded
		}
	}

	if params.Barrier == "" {
		params.Barrier = time.Now().UTC().Add(-2 * time.Second).Format(time.RFC3339)
	}

	where := []string{}

	if params.QueryString != "" {
		tsquery := fts.AppQueryToTsquery(params.QueryString)

		where = append(where, fmt.Sprintf("t.tsv @@ $%d::tsquery", argIdx))
		args = append(args, tsquery)
		argIdx++
	}

	if len(params.InfoHashes) > 0 {
		appendINClause(&where, "t.info_hash", params.InfoHashes, &args, &argIdx)
	}

	var (
		ctWhereIdx int
		ctArgStart int
		ctArgCount int
	)

	if len(params.ContentTypes) > 0 {
		ctWhereIdx = len(where)
		ctArgStart = argIdx
		ctArgCount = len(params.ContentTypes)
		appendINClause(&where, "t.content_type", params.ContentTypes, &args, &argIdx)
	}

	if len(params.TorrentSources) > 0 {
		placeholders := make([]string, len(params.TorrentSources))
		for i, src := range params.TorrentSources {
			placeholders[i] = fmt.Sprintf("$%d", argIdx)

			args = append(args, src)
			argIdx++
		}

		where = append(
			where,
			fmt.Sprintf(
				`EXISTS (SELECT 1 FROM torrents_torrent_sources tts WHERE tts.info_hash = t.info_hash AND tts.source IN (%s))`,
				strings.Join(placeholders, ", "),
			),
		)
	}

	if len(params.FileTypes) > 0 {
		extClauses := make([]string, 0, len(params.FileTypes))

		for _, ft := range params.FileTypes {
			exts := fileTypeExtensions(ft)

			extPlaceholders := make([]string, len(exts))
			for i, ext := range exts {
				extPlaceholders[i] = fmt.Sprintf("$%d", argIdx)

				args = append(args, ext)
				argIdx++
			}

			extClauses = append(
				extClauses,
				fmt.Sprintf(
					"EXISTS (SELECT 1 FROM torrent_files tf WHERE tf.info_hash = t.info_hash AND tf.extension IN (%s))",
					strings.Join(extPlaceholders, ", "),
				),
			)
		}

		if len(extClauses) > 0 {
			where = append(where, "("+strings.Join(extClauses, " OR ")+")")
		}
	}

	if len(params.Languages) > 0 {
		placeholders := make([]string, len(params.Languages))
		for i, lang := range params.Languages {
			placeholders[i] = fmt.Sprintf("$%d", argIdx)

			args = append(args, lang)
			argIdx++
		}

		where = append(where, fmt.Sprintf("t.languages ?| ARRAY[%s]", strings.Join(placeholders, ", ")))
	}

	if len(params.ReleaseYears) > 0 {
		appendINClause(&where, "EXTRACT(YEAR FROM c.release_date)::int", params.ReleaseYears, &args, &argIdx)
	}

	if len(params.ContentRefs) > 0 {
		tupleParts := make([]string, len(params.ContentRefs))
		for i, ref := range params.ContentRefs {
			tupleParts[i] = fmt.Sprintf("($%d, $%d, $%d)", argIdx, argIdx+1, argIdx+2)

			args = append(args, ref.Type, ref.Source, ref.ID)
			argIdx += 3
		}

		where = append(
			where,
			fmt.Sprintf("(t.content_type, t.content_source, t.content_id) IN (%s)", strings.Join(tupleParts, ", ")),
		)
	}

	if params.Barrier != "" {
		where = append(where, fmt.Sprintf("t.created_at <= $%d::timestamptz", argIdx))
		args = append(args, params.Barrier)
		argIdx++
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = "WHERE " + strings.Join(where, "\n  AND ")
	}

	facetWhereClause := whereClause
	facetArgs := args

	if ctArgCount > 0 {
		facetWhere := make([]string, 0, len(where)-1)
		facetWhere = append(facetWhere, where[:ctWhereIdx]...)

		facetWhere = append(facetWhere, where[ctWhereIdx+1:]...)
		if len(facetWhere) > 0 {
			re := regexp.MustCompile(`\$(\d+)`)
			for i, clause := range facetWhere {
				facetWhere[i] = re.ReplaceAllStringFunc(clause, func(match string) string {
					num, _ := strconv.Atoi(match[1:])
					if num > ctArgStart {
						return fmt.Sprintf("$%d", num-ctArgCount)
					}

					return match
				})
			}

			facetWhereClause = "WHERE " + strings.Join(facetWhere, "\n  AND ")
		} else {
			facetWhereClause = ""
		}

		facetArgs = make([]any, 0, len(args)-ctArgCount)
		facetArgs = append(facetArgs, args[:ctArgStart-1]...)
		facetArgs = append(facetArgs, args[ctArgStart-1+ctArgCount:]...)
	}

	offset := utils.ClampInt32(params.Offset)

	q := buildTorrentContentQueries(params, whereClause, args, argIdx, offset)

	rows, err := s.q.Pool().Query(ctx, q.query, q.args...)
	if err != nil {
		return search.TorrentSearchResult{}, fmt.Errorf("query torrent content: %w", err)
	}
	defer rows.Close()

	var items []search.TorrentSearchRow

	for rows.Next() {
		var row search.TorrentSearchRow

		err := rows.Scan(
			&row.InfoHash, &row.ContentType, &row.ContentSource, &row.ContentID,
			&row.Languages,
			&row.Tsv, &row.Seeders, &row.Leechers, &row.Size,
			&row.FilesCount, &row.CreatedAt, &row.UpdatedAt,
			&row.TorrentName, &row.TorrentPrivate,
			&row.ContentTitle, &row.ContentOverview,
			&row.ContentCreatedAt, &row.ContentUpdatedAt,
		)
		if err != nil {
			return search.TorrentSearchResult{}, fmt.Errorf("scan torrent content row: %w", err)
		}

		items = append(items, row)
	}

	if err := rows.Err(); err != nil {
		return search.TorrentSearchResult{}, fmt.Errorf("rows iteration: %w", err)
	}

	result := search.TorrentSearchResult{Items: items}

	// Total count
	if params.TotalCount {
		err := s.q.Pool().QueryRow(ctx, q.countQuery, q.countArgs...).Scan(&result.TotalCount)
		if err != nil {
			return search.TorrentSearchResult{}, fmt.Errorf("count torrent content: %w", err)
		}
	}

	// Has next page
	if params.HasNextPage && len(items) > 0 {
		var nextExists bool

		err := s.q.Pool().QueryRow(ctx, q.hasNextQuery, q.hasNextArgs...).Scan(&nextExists)
		if err == nil {
			result.HasNextPage = nextExists
		}
	}

	// Facet aggregations
	aggs, err := s.computeTorrentContentFacets(ctx, facetWhereClause, facetArgs, params)
	if err != nil {
		return search.TorrentSearchResult{}, fmt.Errorf("facet aggregations: %w", err)
	}

	result.Aggregations = aggs

	result.Barrier = search.EncodeBarrier(params.Barrier)

	return result, nil
}

func buildTorrentContentOrderBy(params search.TorrentSearchParams, args *[]any, argIdx *int) string {
	if len(params.OrderBy) == 0 {
		return defaultTorrentContentOrderBy(params, args, argIdx)
	}

	clauses := []string{}

	for i, ob := range params.OrderBy {
		isLast := i == len(params.OrderBy)-1

		dir := "ASC"
		if ob.Direction == search.SortDesc {
			dir = "DESC"
		}

		var clause string

		switch ob.Field {
		case search.FieldRelevance:
			if params.QueryString != "" {
				tsquery := fts.AppQueryToTsquery(params.QueryString)
				clause = fmt.Sprintf("ts_rank(t.tsv, $%d::tsquery) %s", *argIdx, dir)

				*args = append(*args, tsquery)
				*argIdx++
			} else {
				clause = fmt.Sprintf("t.created_at %s", dir)
			}
		case search.FieldCreatedAt:
			clause = fmt.Sprintf("t.created_at %s", dir)
		case search.FieldUpdatedAt:
			clause = fmt.Sprintf("t.updated_at %s", dir)
		case search.FieldSize:
			clause = fmt.Sprintf("t.size %s", dir)
		case search.FieldFilesCount:
			clause = fmt.Sprintf("COALESCE(t.files_count, 0) %s", dir)
		case search.FieldSeeders:
			clause = fmt.Sprintf("COALESCE(t.seeders, -1) %s", dir)
		case search.FieldLeechers:
			clause = fmt.Sprintf("COALESCE(t.leechers, -1) %s", dir)
		case search.FieldName:
			clause = fmt.Sprintf("t.name %s", dir)
		case search.FieldInfoHash:
			clause = fmt.Sprintf("t.info_hash %s", dir)
		}

		if clause != "" {
			if isLast {
				clause += fmt.Sprintf(", t.info_hash %s", dir)
			}

			clauses = append(clauses, clause)
		}
	}

	if len(clauses) == 0 {
		return defaultTorrentContentOrderBy(params, args, argIdx)
	}

	return "ORDER BY " + strings.Join(clauses, ", ")
}

func (s *pgSearch) computeTorrentContentFacets(
	ctx context.Context,
	whereClause string,
	args []any,
	params search.TorrentSearchParams,
) (map[string]search.AggregationBucket, error) {
	aggs := map[string]search.AggregationBucket{}

	if params.FacetAggregate.ContentType {
		bucket, err := s.simpleFacet(ctx, `t.content_type`, whereClause, args)
		if err != nil {
			return nil, fmt.Errorf("content type facet: %w", err)
		}

		aggs["content_type"] = bucket
	}

	if params.FacetAggregate.TorrentSource {
		bucket, err := s.existsFacet(ctx, `torrents_torrent_sources`, `source`, `info_hash`, `t.info_hash`, whereClause, args)
		if err != nil {
			return nil, fmt.Errorf("torrent source facet: %w", err)
		}

		aggs["torrent_source"] = bucket
	}

	if params.FacetAggregate.FileType {
		bucket, err := s.fileTypeFacet(ctx, whereClause, args)
		if err != nil {
			return nil, fmt.Errorf("file type facet: %w", err)
		}

		aggs["file_type"] = bucket
	}

	if params.FacetAggregate.Language {
		bucket, err := s.jsonbFacet(ctx, `t.languages`, whereClause, args)
		if err != nil {
			return nil, fmt.Errorf("language facet: %w", err)
		}

		aggs["language"] = bucket
	}

	if params.FacetAggregate.ReleaseYear {
		bucket, err := s.simpleFacet(ctx, `EXTRACT(YEAR FROM c.release_date)::int`, whereClause, args)
		if err != nil {
			return nil, fmt.Errorf("release year facet: %w", err)
		}

		aggs["release_year"] = bucket
	}

	return aggs, nil
}

func (s *pgSearch) simpleFacet(ctx context.Context, column string, whereClause string, args []any) (search.AggregationBucket, error) {
	from := "torrents t"
	if strings.Contains(column, "c.") {
		from += " LEFT JOIN content c ON t.content_type = c.type AND t.content_source = c.source AND t.content_id = c.id"
	}

	query := fmt.Sprintf(
		`SELECT %s AS value, COUNT(*) AS count FROM %s %s GROUP BY %s ORDER BY count DESC, value ASC`,
		column,
		from,
		whereClause,
		column,
	)

	return s.queryFacet(ctx, query, args)
}

func (s *pgSearch) existsFacet(
	ctx context.Context,
	table, valueCol, keyCol, joinExpr, whereClause string,
	args []any,
) (search.AggregationBucket, error) {
	query := fmt.Sprintf(`SELECT tt.%s AS value, COUNT(*) AS count FROM
		(SELECT DISTINCT %s.%s FROM %s, LATERAL (SELECT DISTINCT %s FROM %s WHERE %s.%s = %s) tt) sub
		CROSS JOIN LATERAL (
			SELECT COUNT(*) FROM torrents t
			LEFT JOIN %s ON %s.%s = %s
			WHERE %s.%s = tt.value AND %s
		) cnt
		ORDER BY count DESC, value ASC`,
		table, table, valueCol,
		table, valueCol, table, table, keyCol, joinExpr,
		table, table, keyCol, joinExpr,
		table, valueCol, whereClause)

	return s.queryFacet(ctx, query, args)
}

func (s *pgSearch) jsonbFacet(ctx context.Context, column, whereClause string, args []any) (search.AggregationBucket, error) {
	query := fmt.Sprintf(`SELECT lang AS value, COUNT(*) AS count FROM
		torrents t
		LEFT JOIN LATERAL jsonb_array_elements_text(%s) AS lang ON true
		%s
		GROUP BY lang ORDER BY count DESC, lang ASC`, column, whereClause)

	return s.queryFacet(ctx, query, args)
}

func (s *pgSearch) fileTypeFacet(ctx context.Context, whereClause string, args []any) (search.AggregationBucket, error) {
	fileTypes := []string{"video", "audio", "image", "archive", "subtitle", "document", "metadata", "other"}
	typeClauses := []string{}
	typeArgs := []any{}
	idx := len(args) + 1

	for _, ft := range fileTypes {
		exts := fileTypeExtensions(ft)
		if len(exts) == 0 {
			continue
		}

		phs := make([]string, len(exts))
		for i, ext := range exts {
			phs[i] = fmt.Sprintf("$%d", idx)

			typeArgs = append(typeArgs, ext)
			idx++
		}

		typeClauses = append(
			typeClauses,
			fmt.Sprintf(
				`EXISTS (SELECT 1 FROM torrent_files tf WHERE tf.info_hash = t.info_hash AND tf.extension IN (%s))`,
				strings.Join(phs, ", "),
			),
		)
	}

	allArgs := slices.Concat(args, typeArgs)

	caseClauses := make([]string, 0, len(fileTypes))
	for i, ft := range fileTypes {
		caseClauses = append(caseClauses, fmt.Sprintf("WHEN %s THEN '%s'", typeClauses[i], ft))
	}

	query := fmt.Sprintf(`SELECT ft.value, COUNT(*) AS count FROM
		torrents t
		CROSS JOIN LATERAL (VALUES %s) ft(value)
		WHERE (CASE %s END) = ft.value
		AND EXISTS (SELECT 1 FROM torrent_files tf2 WHERE tf2.info_hash = t.info_hash)
		AND %s
		GROUP BY ft.value ORDER BY count DESC`,
		strings.Join(func() []string {
			vs := make([]string, len(fileTypes))
			for i, ft := range fileTypes {
				vs[i] = fmt.Sprintf("('%s')", ft)
			}

			return vs
		}(), ", "),
		strings.Join(caseClauses, " "),
		whereClause)

	return s.queryFacet(ctx, query, allArgs)
}

func (s *pgSearch) queryFacet(ctx context.Context, query string, args []any) (search.AggregationBucket, error) {
	bucket := search.AggregationBucket{Items: map[string]search.AggregationItem{}}

	rows, err := s.q.Pool().Query(ctx, query, args...)
	if err != nil {
		return bucket, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			value *string
			count int64
		)
		if err := rows.Scan(&value, &count); err != nil {
			return bucket, err
		}

		if value != nil {
			bucket.Items[*value] = search.AggregationItem{Count: uint(count), Label: *value}
		}
	}

	if err := rows.Err(); err != nil {
		return bucket, err
	}

	return bucket, nil
}

func appendINClause[T any](where *[]string, col string, items []T, args *[]any, argIdx *int) {
	if len(items) == 0 {
		return
	}

	placeholders := make([]string, len(items))
	for i, item := range items {
		placeholders[i] = fmt.Sprintf("$%d", *argIdx)

		*args = append(*args, item)
		*argIdx++
	}

	*where = append(*where, fmt.Sprintf("%s IN (%s)", col, strings.Join(placeholders, ", ")))
}

func defaultTorrentContentOrderBy(params search.TorrentSearchParams, args *[]any, argIdx *int) string {
	if params.QueryString != "" {
		tsquery := fts.AppQueryToTsquery(params.QueryString)
		*args = append(*args, tsquery)
		result := fmt.Sprintf("ORDER BY ts_rank(t.tsv, $%d::tsquery) DESC, t.info_hash DESC", *argIdx)
		*argIdx++

		return result
	}

	return `ORDER BY t.created_at DESC, t.info_hash DESC`
}

func fileTypeExtensions(fileType string) []string {
	switch fileType {
	case "video":
		return []string{"avi", "divx", "m2ts", "m4v", "mkv", "mov", "mp4", "mpeg", "mpg", "ts", "webm", "wmv", "vob", "iso"}
	case "audio":
		return []string{"aac", "ac3", "ape", "dts", "flac", "m4a", "mp3", "ogg", "opus", "wav", "wma"}
	case "image":
		return []string{"bmp", "gif", "jpg", "jpeg", "png", "tbn", "webp"}
	case "archive":
		return []string{"7z", "bz2", "gz", "lzma", "rar", "tar", "xz", "zst", "zip", "zpaq"}
	case "subtitle":
		return []string{"ass", "idx", "srt", "ssa", "sub", "sup", "vtt"}
	case "document":
		return []string{"doc", "docx", "htm", "html", "mht", "mobi", "pdf", "txt", "xls", "xlsx", "xml"}
	case "metadata":
		return []string{"nfo", "sfv"}
	default:
		return nil
	}
}

var torrentContentSelectCols = `
		t.info_hash, t.content_type, t.content_source, t.content_id,
		t.languages,
		t.tsv, t.seeders, t.leechers, t.size,
		t.files_count, t.created_at, t.updated_at,
		t.name AS t_name, t.private AS t_private,
		c.title AS c_title, c.overview AS c_overview,
		c.created_at AS c_created_at, c.updated_at AS c_updated_at`

type torrentContentQuery struct {
	query        string
	countQuery   string
	countArgs    []any
	hasNextQuery string
	hasNextArgs  []any
	args         []any
}

func buildTorrentContentQueries(
	params search.TorrentSearchParams,
	whereClause string,
	whereArgs []any,
	inputArgIdx int,
	offset int32,
) torrentContentQuery {
	argIdx := inputArgIdx
	args := whereArgs
	preOrderByArgsLen := len(args)
	orderByClause := buildTorrentContentOrderBy(params, &args, &argIdx)

	limit := utils.ClampInt32(params.Limit)
	if limit <= 0 {
		limit = 10
	}

	query := fmt.Sprintf(`SELECT %s FROM torrents t
		LEFT JOIN content c ON t.content_type = c.type AND t.content_source = c.source AND t.content_id = c.id
		%s
		%s
		LIMIT $%d OFFSET $%d`,
		torrentContentSelectCols, whereClause, orderByClause, argIdx, argIdx+1)

	offsetArg := argIdx + 1

	args = append(args, limit, offset)
	argIdx += 2

	var (
		countQuery string
		countArgs  []any
	)

	if params.TotalCount {
		countQuery = fmt.Sprintf(`SELECT COUNT(*) FROM torrents t
			LEFT JOIN content c ON t.content_type = c.type AND t.content_source = c.source AND t.content_id = c.id
			%s`, whereClause)
		countArgs = args[:preOrderByArgsLen]
	}

	var (
		hasNextQuery string
		hasNextArgs  []any
	)

	if params.HasNextPage {
		hasNextQuery = fmt.Sprintf(`SELECT EXISTS(SELECT 1 FROM torrents t
			LEFT JOIN content c ON t.content_type = c.type AND t.content_source = c.source AND t.content_id = c.id
			%s
			%s OFFSET $%d)`,
			whereClause, orderByClause, offsetArg-1)

		hasNextArgs = append(slices.Clone(args[:len(args)-2]), offset+limit)
	}

	return torrentContentQuery{
		query:        query,
		countQuery:   countQuery,
		countArgs:    countArgs,
		hasNextQuery: hasNextQuery,
		hasNextArgs:  hasNextArgs,
		args:         args,
	}
}
