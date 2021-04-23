package main

import (
	"net/http"
	"strconv"
)

func resourceHandler(w http.ResponseWriter, req *http.Request) {
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

	p, err := bru.GetPost(req.Context(), id)
	if err != nil {
		errorHandler(w, req, err)
		return
	}

	http.ServeFile(w, req, p.Post)
}
