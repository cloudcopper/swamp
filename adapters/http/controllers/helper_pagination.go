package controllers

import (
	"net/http"
	"strconv"
)

func helperPagination[T any](r *http.Request, data []T, perPage int) (_ []T, page int) {
	page = 1
	if r.URL.Query().Get("page") != "" {
		page, _ = strconv.Atoi(r.URL.Query().Get("page"))
	}
	if page < 1 {
		page = 1
	}

	total := len(data)
	pages := (total + perPage - 1) / perPage
	if pages < 1 {
		return data, 1
	}
	if page > pages {
		page = pages
	}
	start := (page - 1) * perPage
	end := start + perPage
	if end > total {
		end = total
	}
	data = data[start:end]
	return data, page
}
