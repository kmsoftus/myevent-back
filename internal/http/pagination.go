package http

import (
	nethttp "net/http"
	"strconv"
)

func ReadPagination(r *nethttp.Request) (int, int) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	return page, pageSize
}
