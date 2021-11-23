package main

import (
	"database/sql"
	"embed"
	"flag"
	"html/template"
	"net/http"

	"github.com/dhlk/booru"
	_ "github.com/mattn/go-sqlite3"
)

//go:embed *.tmpl
var templatesFS embed.FS

//go:embed *.css
var stylesFS embed.FS

var (
	templates *template.Template
	bru       *booru.Booru

	btitle = flag.String("title", "ssbooru", "title")

	address  = flag.String("address", "localhost:8085", "address to listen on")
	dbpath   = flag.String("db", "booru.db", "sqlite3 database")
	index    = flag.String("index", "index", "index directory")
	baseline = flag.String("baseline", "baseline", "baseline directory")
)

func main() {
	flag.Parse()

	var err error
	templates, err = template.ParseFS(templatesFS, "*.tmpl")
	if err != nil {
		panic(err)
	}

	var db *sql.DB
	db, err = sql.Open("sqlite3", *dbpath)
	if err != nil {
		panic(err)
	}

	bru = booru.New(db, *index, *baseline)

	http.Handle("/post/", http.StripPrefix("/post/", http.HandlerFunc(postHandler)))
	http.Handle("/resource/", http.StripPrefix("/resource/", http.HandlerFunc(resourceHandler)))
	http.Handle("/styles/", http.StripPrefix("/styles/", http.FileServer(http.FS(stylesFS))))
	http.HandleFunc("/index", indexHandler)
	http.HandleFunc("/search", searchHandler)
	http.HandleFunc("/length", lengthHandler)

	panic(http.ListenAndServe(*address, nil))
}
