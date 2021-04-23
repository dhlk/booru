package main

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/dhlk/booru"
)

var (
	errorBadID = errors.New("Bad post id.")
)

type PostPage struct {
	Base BasePage
	Post booru.Post
	Edit bool
}

func postHandler(w http.ResponseWriter, req *http.Request) {
	req.ParseForm()

	if req.Form["id"] == nil || len(req.Form["id"]) != 1 {
		errorHandler(w, req, errorBadID)
		return
	}

	id, err := strconv.ParseInt(req.Form["id"][0], 10, 64)
	if err != nil {
		errorHandler(w, req, err)
		return
	}

	// handle tagging
	if req.Form["tag"] != nil {
		for _, tag := range req.Form["tag"] {
			err = bru.TagPost(req.Context(), id, tag)
			if err != nil {
				errorHandler(w, req, err)
				return
			}
		}

		url := req.URL
		query := url.Query()
		query["tag"] = nil
		url.RawQuery = query.Encode()

		http.Redirect(w, req, url.RequestURI(), 302)
	}

	// handle untagging
	if req.Form["untag"] != nil && len(req.Form["untag"]) == 1 {
		for _, tag := range req.Form["untag"] {
			err = bru.UnTagPost(req.Context(), id, tag)
			if err != nil {
				errorHandler(w, req, err)
				return
			}
		}

		url := req.URL
		query := url.Query()
		query["untag"] = nil
		url.RawQuery = query.Encode()

		http.Redirect(w, req, url.RequestURI(), 302)
	}

	p, err := bru.GetPost(req.Context(), id)
	if err != nil {
		errorHandler(w, req, err)
		return
	}

	post := PostPage{
		Base: NewBasePage(),
		Post: p,
	}

	templates.ExecuteTemplate(w, "post.tmpl", post)
}
