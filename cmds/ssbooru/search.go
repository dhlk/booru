package main

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/dhlk/booru"
)

var (
	errorBadQuery = errors.New("Bad query.")
)

type SearchPage struct {
	Base   BasePage
	M3u    bool
	Direct bool
	Query  string
	Page   int64
	Prev   int64
	Next   int64
	Length int64
	Posts  []booru.Post
	Tags   []booru.Tag
}

func searchHandler(w http.ResponseWriter, req *http.Request) {
	req.ParseForm()

	var err error
	query := ""
	page := int64(0)
	length := int64(24)

	isM3u := req.Form["m3u"] != nil
	direct := req.Form["direct"] != nil

	if req.Form["query"] != nil {
		if len(req.Form["query"]) != 1 {
			errorHandler(w, req, errorBadQuery)
			return
		}
		query = req.Form["query"][0]
	}

	if req.Form["page"] != nil {
		if len(req.Form["page"]) != 1 {
			errorHandler(w, req, errorBadQuery)
			return
		}
		page, err = strconv.ParseInt(req.Form["page"][0], 10, 64)
		if err != nil {
			errorHandler(w, req, err)
			return
		}
	}

	if req.Form["length"] != nil {
		if len(req.Form["length"]) != 1 {
			errorHandler(w, req, errorBadQuery)
			return
		}
		length, err = strconv.ParseInt(req.Form["length"][0], 10, 64)
		if err != nil {
			errorHandler(w, req, err)
			return
		}
	}

	posts, err := bru.Query(req.Context(), query, page, length)
	if err != nil {
		errorHandler(w, req, err)
		return
	}

	next := page + 1
	if length != int64(len(posts)) {
		next = -1
	}

	search := SearchPage{
		Base:   NewBasePage(),
		M3u:    isM3u,
		Direct: direct,
		Query:  query,
		Page:   page,
		Prev:   page - 1,
		Next:   next,
		Length: length,
		Posts:  posts,
		Tags:   []booru.Tag{},
	}

	templates.ExecuteTemplate(w, "search.tmpl", search)
}
