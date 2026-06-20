package gqlmodel

import (
	"errors"
	"fmt"
	"sort"

	"github.com/facette/natsort"
	"github.com/hexsans/hexmagnet/internal/gql/gqlmodel/gen"
	"github.com/hexsans/hexmagnet/internal/model"
	dbsearch "github.com/hexsans/hexmagnet/internal/search"
)

func aggs[T any, Agg comparable](
	items dbsearch.AggregationItems,
	parse func(string) (T, error),
	newAgg func(value *T, label string, count uint, isEstimate bool) Agg,
) ([]Agg, error) {
	if items == nil {
		return nil, nil
	}

	r := make([]Agg, 0, len(items))
	labelMap := make(map[Agg]string, len(items))

	for key, item := range items {
		if key != "null" {
			v, err := parse(key)
			if err != nil {
				return nil, fmt.Errorf("error parsing aggregation item: %w", err)
			}

			agg := newAgg(&v, item.Label, item.Count, item.IsEstimate)
			r = append(r, agg)
			labelMap[agg] = item.Label
		}
	}

	sort.Slice(r, func(i, j int) bool {
		return natsort.Compare(labelMap[r[i]], labelMap[r[j]])
	})

	if null, nullOk := items["null"]; nullOk {
		r = append(r, newAgg(nil, null.Label, null.Count, null.IsEstimate))
	}

	return r, nil
}

func contentTypeAggs(items dbsearch.AggregationItems) ([]gen.ContentTypeAgg, error) {
	return aggs(items, model.ParseContentType,
		func(value *model.ContentType, label string, count uint, isEstimate bool) gen.ContentTypeAgg {
			return gen.ContentTypeAgg{Value: value, Label: label, Count: int(count), IsEstimate: isEstimate}
		})
}

func torrentFileTypeAggs(items dbsearch.AggregationItems) ([]gen.TorrentFileTypeAgg, error) {
	return aggs(items, model.ParseFileType,
		func(value *model.FileType, label string, count uint, isEstimate bool) gen.TorrentFileTypeAgg {
			return gen.TorrentFileTypeAgg{
				Value:      *value,
				Label:      label,
				Count:      int(count),
				IsEstimate: isEstimate,
			}
		})
}

func languageAggs(items dbsearch.AggregationItems) ([]gen.LanguageAgg, error) {
	return aggs(items, func(str string) (model.Language, error) {
		lang := model.ParseLanguage(str)
		if !lang.Valid {
			return "", errors.New("invalid language")
		}

		return lang.Language, nil
	}, func(value *model.Language, label string, count uint, isEstimate bool) gen.LanguageAgg {
		return gen.LanguageAgg{Value: *value, Label: label, Count: int(count), IsEstimate: isEstimate}
	})
}

func releaseYearAggs(items dbsearch.AggregationItems) ([]gen.ReleaseYearAgg, error) {
	return aggs(items, model.ParseYear,
		func(value *model.Year, label string, count uint, isEstimate bool) gen.ReleaseYearAgg {
			return gen.ReleaseYearAgg{Value: value, Label: label, Count: int(count), IsEstimate: isEstimate}
		})
}
