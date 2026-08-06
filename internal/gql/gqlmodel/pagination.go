package gqlmodel

import "github.com/hexsans/hexmagnet/internal/model"

type PaginationParams struct {
	Limit       uint
	Offset      uint
	TotalCount  bool
	HasNextPage bool
}

func ExtractPagination(
	limit model.NullUint,
	page model.NullUint,
	offset model.NullUint,
	totalCount model.NullBool,
	hasNextPage model.NullBool,
) PaginationParams {
	p := PaginationParams{Limit: 10}

	if limit.Valid {
		p.Limit = limit.Uint
	}

	if page.Valid && page.Uint > 0 {
		p.Offset = (page.Uint - 1) * p.Limit
	}

	if offset.Valid {
		p.Offset += offset.Uint
	}

	p.TotalCount = totalCount.Valid && totalCount.Bool
	p.HasNextPage = hasNextPage.Valid && hasNextPage.Bool

	return p
}
