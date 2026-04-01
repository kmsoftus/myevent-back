package services

const (
	defaultPageSize = 100
	maxPageSize     = 100
)

type Pagination struct {
	Page     int
	PageSize int
	Offset   int
}

type PagedResult[T any] struct {
	Items      []T
	Total      int
	Page       int
	PageSize   int
	TotalPages int
}

func normalizePagination(page, pageSize int) Pagination {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	return Pagination{
		Page:     page,
		PageSize: pageSize,
		Offset:   (page - 1) * pageSize,
	}
}

func totalPages(total, pageSize int) int {
	if total == 0 || pageSize <= 0 {
		return 0
	}

	return (total + pageSize - 1) / pageSize
}
