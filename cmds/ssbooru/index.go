package main

import (
	"fmt"
	"log"
	"net/http"
)

func indexHandler(w http.ResponseWriter, req *http.Request) {
	log.Printf("ssbooru recieved index-regen command")
	defer req.Body.Close()
	err := bru.GenerateIndexes(req.Context())
	log.Printf("index regen complete with error %v", err)
	fmt.Fprintf(w, "%v", err)
}
