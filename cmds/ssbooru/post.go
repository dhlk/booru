package main

import (
	"errors"
	"net/http"

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

	p, err := bru.GetPost(req.Context(), req.URL.Path)
	if err != nil {
		p = booru.Post{
			ID:   -1,
			Post: req.URL.Path,
			Tags: []booru.Tag{
				{Tag: "not in db"},
				{Tag: err.Error()},
			},
		}
		//errorHandler(w, req, err)
		//return
	}

	post := PostPage{
		Base: NewBasePage(),
		Post: p,
	}

	templates.ExecuteTemplate(w, "post.tmpl", post)
}
