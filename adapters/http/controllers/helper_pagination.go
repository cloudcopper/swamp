package controllers

import (
	"net/http"
	"strconv"
)

func helperPagination[T any](r *http.Request, data []T, perPage int) ([]T, int) {
	page := 1
	if r.URL.Query().Get("page") != "" {
		page, _ = strconv.Atoi(r.URL.Query().Get("page"))
	}
	total := len(data)
	pages := (total + perPage - 1) / perPage
	if page < 1 {
		page = 1
	} else if page > pages {
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
