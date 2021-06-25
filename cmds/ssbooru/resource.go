package main

import "net/http"

func resourceHandler(w http.ResponseWriter, req *http.Request) {
	req.ParseForm()

	p, err := bru.GetPost(req.Context(), req.URL.Path)
	if err != nil {
		errorHandler(w, req, err)
		return
	}

	http.ServeFile(w, req, p.Post)
}
