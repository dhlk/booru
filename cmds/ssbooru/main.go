package main

import (
	"database/sql"
	"flag"
	"html/template"
	"net/http"

	"github.com/dhlk/booru"
	_ "github.com/mattn/go-sqlite3"
)

var (
	templates *template.Template
	bru       *booru.Booru

	btitle = flag.String("title", "ssbooru", "title")

	address  = flag.String("address", "localhost:8085", "address to listen on")
	dbpath   = flag.String("db", "booru.db", "sqlite3 database")
	stpath   = flag.String("styles", "./styles", "sqlite3 database")
	tmglob   = flag.String("tmpl", "./pages/*.tmpl", "template glob")
	index    = flag.String("index", "index", "index directory")
	baseline = flag.String("baseline", "baseline", "baseline directory")
)

func main() {
	flag.Parse()

	var err error
	templates, err = template.ParseGlob(*tmglob)
	if err != nil {
		panic(err)
	}

	var db *sql.DB
	db, err = sql.Open("sqlite3", *dbpath)
	if err != nil {
		panic(err)
	}

	bru = booru.New(db, *index, *baseline)

	http.Handle("/styles/", http.StripPrefix("/styles/", http.FileServer(http.Dir(*stpath))))
	http.HandleFunc("/post", postHandler)
	http.HandleFunc("/search", searchHandler)
	http.HandleFunc("/resource", resourceHandler)
	http.HandleFunc("/index", indexHandler)

	panic(http.ListenAndServe(*address, nil))
}
