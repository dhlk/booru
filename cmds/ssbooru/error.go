package main

import (
	"net/http"
)

type ErrorPage struct {
	Base  BasePage
	Error error
}

func errorHandler(w http.ResponseWriter, req *http.Request, err error) {
	ep := ErrorPage{
		Base:  NewBasePage(),
		Error: err,
	}

	templates.ExecuteTemplate(w, "error.tmpl", ep)
}
