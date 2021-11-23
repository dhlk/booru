package main

import (
	"fmt"
	"net/http"
)

func lengthHandler(w http.ResponseWriter, req *http.Request) {
	req.ParseForm()

	var err error
	query := ""
	if req.Form["query"] != nil {
		if len(req.Form["query"]) != 1 {
			errorHandler(w, req, errorBadQuery)
			return
		}
		query = req.Form["query"][0]
	}

	length, err := bru.Count(req.Context(), query)
	if err != nil {
		errorHandler(w, req, err)
		return
	}

	w.Header().Set("Content-Type", "text/json")
	fmt.Fprintf(w, "%d", length)
}
