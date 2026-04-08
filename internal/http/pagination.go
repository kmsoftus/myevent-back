package http

import (
	nethttp "net/http"
	"strconv"
	"strings"
)

func ReadPagination(r *nethttp.Request) (int, int) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	return page, pageSize
}

func HasPaginationParams(r *nethttp.Request) bool {
	query := r.URL.Query()
	return strings.TrimSpace(query.Get("page")) != "" ||
		strings.TrimSpace(query.Get("page_size")) != ""
}
