package paginator

import (
	"context"
	"fmt"

	"github.com/paulexconde/justasking/internal/pkg/store"
)

type PaginatedResponse[T any] struct {
	Items       []T  `json:"items"`
	CurrentPage int  `json:"current_page"`
	TotalPages  int  `json:"total_pages"`
	PrevPage    *int `json:"prev_page"`
	NextPage    *int `json:"next_page"`
	TotalItems  int  `json:"total_items"`
}

type Paginator[T any] interface {
	// Pagination based from custom query.
	PaginateQuery(ctx context.Context, query string, args []any, page, limit int) (*PaginatedResponse[T], error)
}

type paginatorImpl[T any] struct {
	datastore store.Datastorer[T]
}

func NewPaginator[T any](ds store.Datastorer[T]) Paginator[T] {
	return &paginatorImpl[T]{datastore: ds}
}

func (p *paginatorImpl[T]) PaginateQuery(ctx context.Context, query string, args []any, page, limit int) (*PaginatedResponse[T], error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}

	offset := (page - 1) * limit

	// Count total rows using a subquery
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM (%s) AS total_count", query)
	totalItemsRaw, err := p.datastore.QueryRow(ctx, countQuery, args...)
	if err != nil {
		return nil, err
	}

	// Type assertion with proper conversion
	var totalItems int
	switch v := totalItemsRaw.(type) {
	case int:
		totalItems = v
	case int64:
		totalItems = int(v) // Convert int64 to int
	default:
		return nil, fmt.Errorf("expected int for total count, got %T", totalItemsRaw)
	}

	totalPages := (totalItems + limit - 1) / limit

	// Append LIMIT & OFFSET to the query
	paginatedQuery := fmt.Sprintf("%s LIMIT $%d OFFSET $%d", query, len(args)+1, len(args)+2)
	args = append(args, limit, offset)

	// Execute the paginated query
	items, err := p.datastore.Select(ctx, paginatedQuery, args...)
	if err != nil {
		return nil, err
	}

	// Determine prev/next pages
	var prevPage, nextPage *int
	if page > 1 {
		p := page - 1
		prevPage = &p
	}
	if page < totalPages {
		p := page + 1
		nextPage = &p
	}

	return &PaginatedResponse[T]{
		Items:       items,
		CurrentPage: page,
		TotalPages:  totalPages,
		PrevPage:    prevPage,
		NextPage:    nextPage,
		TotalItems:  totalItems,
	}, nil
}
