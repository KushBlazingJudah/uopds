package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

var (
	root, coverDir, bookDir string
)

func main() {
	flag.StringVar(&root, "root", "", "root directory for the http server")
	flag.StringVar(&coverDir, "covers", "covers", "directory for cover images")
	flag.StringVar(&bookDir, "books", "books", "directory for books")

	flag.Parse()

	rootFeed := feed{
		Id: &uuidurn{},
		Links: []link{
			{Rel: "self", Href: root + "/", Type: "application/atom+xml;profile=opds-catalog;kind=navigation"},
			{Rel: "start", Href: root + "/", Type: "application/atom+xml;profile=opds-catalog;kind=navigation"},
		},
		Title:   "uopds",
		Updated: time.Now(),
		Author: author{
			Name: "uopds",
			URI:  &uuidurn{},
		},
		Entries: []entry{},
	}

	// generate entries in new catalog
	dir, err := os.ReadDir(bookDir)
	if err != nil {
		panic(err)
	}

	for _, file := range dir {
		if !file.Type().IsRegular() || filepath.Ext(file.Name()) != ".epub" {
			// ignore it
			continue
		}

		// generate an entry for it
		opf, err := readOpfFromEpub(file.Name())
		if err != nil {
			panic(err)
		}

		e, err := opf.genEntry()
		if err != nil {
			panic(err)
		}

		rootFeed.Entries = append(rootFeed.Entries, e)
	}

	http.HandleFunc(root+"/", func(w http.ResponseWriter, r *http.Request) {
		out, err := xml.Marshal(rootFeed)
		if err != nil {
			w.WriteHeader(503)
			fmt.Fprint(w, err)
			return
		}

		w.Write(out)
	})

	http.Handle(root+"/covers/", http.StripPrefix(root+"/covers/", http.FileServer(http.Dir(coverDir))))
	http.Handle(root+"/books/", http.StripPrefix(root+"/books/", http.FileServer(http.Dir(bookDir))))

	log.Fatal(http.ListenAndServe(":8080", nil))
}
