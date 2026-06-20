package classifier

import (
	"context"
	"fmt"

	"github.com/hexsans/hexmagnet/internal/database/db"
	"github.com/hexsans/hexmagnet/internal/database/fts"
	"github.com/hexsans/hexmagnet/internal/model"
	"go.uber.org/zap"
)

type LocalSearch interface {
	ContentByID(context.Context, model.ContentRef) (model.Content, error)
	ContentBySearch(context.Context, model.ContentType, string, model.Year) (model.Content, error)
}

type localSearch struct {
	q      *db.Queries
	logger *zap.SugaredLogger
}

func (l localSearch) ContentByID(ctx context.Context, ref model.ContentRef) (model.Content, error) {
	content, err := l.q.GetContent(ctx, db.GetContentParams{
		Type:   string(ref.Type),
		Source: ref.Source,
		ID:     ref.ID,
	})
	if err != nil {
		l.logger.Debugw("content by id failed",
			"type", ref.Type, "source", ref.Source, "id", ref.ID, "error", err,
		)

		return model.Content{}, ErrUnmatched
	}

	return dbContentToModel(content), nil
}

func (l localSearch) ContentBySearch(
	ctx context.Context,
	ct model.ContentType,
	baseTitle string,
	year model.Year,
) (model.Content, error) {
	tsquery := fts.AppQueryToTsquery(baseTitle)

	rows, err := l.q.Read(ctx).Query(ctx, `
		SELECT type, source, id, title, release_date, adult,
			overview,
			popularity, vote_average, vote_count, tsv, created_at, updated_at,
			ts_rank(tsv, $1::tsquery) AS rank
		FROM content
		WHERE type = $2
			AND tsv @@ $1::tsquery
			AND ($3::int IS NULL OR EXTRACT(YEAR FROM release_date)::int = $3)
		ORDER BY rank DESC
		LIMIT 10
	`, tsquery, string(ct), yearPtr(year))
	if err != nil {
		return model.Content{}, fmt.Errorf("search query '%s' (%s, %d): %w", baseTitle, ct, year, err)
	}
	defer rows.Close()

	var items []model.Content

	for rows.Next() {
		var (
			c    db.Content
			rank float64
		)
		if err := rows.Scan(
			&c.Type, &c.Source, &c.ID, &c.Title,
			&c.ReleaseDate, &c.Adult,
			&c.Overview,
			&c.Popularity, &c.VoteAverage, &c.VoteCount,
			&c.Tsv, &c.CreatedAt, &c.UpdatedAt, &rank,
		); err != nil {
			return model.Content{}, fmt.Errorf("scan '%s': %w", baseTitle, err)
		}

		items = append(items, dbContentToModel(c))
	}

	if err := rows.Err(); err != nil {
		return model.Content{}, fmt.Errorf("rows '%s': %w", baseTitle, err)
	}

	if len(items) == 0 {
		return model.Content{}, ErrUnmatched
	}

	bestMatch, ok := levenshteinFindBestMatch[model.Content](
		baseTitle, items,
		func(item model.Content) []string {
			return []string{item.Title}
		},
	)
	if !ok {
		return model.Content{}, ErrUnmatched
	}

	return bestMatch, nil
}

func dbContentToModel(c db.Content) model.Content {
	tsv, _ := fts.ParseTsvector(c.Tsv)

	return model.Content{
		Type:        model.ContentType(c.Type),
		Source:      c.Source,
		ID:          c.ID,
		Title:       c.Title,
		ReleaseDate: model.NewDateFromTime(c.ReleaseDate.Time),
		Adult:       boolPtrToNullBool(c.Adult),
		Overview:    strPtrToNullString(c.Overview),
		Popularity:  float64PtrToNullFloat32(c.Popularity),
		VoteAverage: float64PtrToNullFloat32(c.VoteAverage),
		VoteCount:   int64PtrToNullUint(c.VoteCount),
		Tsv:         tsv,
		CreatedAt:   c.CreatedAt.Time,
		UpdatedAt:   c.UpdatedAt.Time,
	}
}

func yearPtr(y model.Year) *int32 {
	if y == 0 {
		return nil
	}

	v := int32(y)

	return &v
}

func boolPtrToNullBool(v *bool) model.NullBool {
	if v == nil {
		return model.NullBool{}
	}

	return model.NewNullBool(*v)
}

func strPtrToNullString(v *string) model.NullString {
	if v == nil {
		return model.NullString{}
	}

	return model.NewNullString(*v)
}

func float64PtrToNullFloat32(v *float64) model.NullFloat32 {
	if v == nil {
		return model.NullFloat32{}
	}

	return model.NewNullFloat32(float32(*v))
}

func int64PtrToNullUint(v *int64) model.NullUint {
	if v == nil {
		return model.NullUint{}
	}

	return model.NewNullUint(uint(*v))
}
