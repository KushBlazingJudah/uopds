package main

import (
	"context"
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
	addr, root, coverDir, bookDir, dbPath string
	db                                    *database
)

var rootFeed feed

func main() {
	flag.StringVar(&addr, "addr", ":8080", "listen address")
	flag.StringVar(&dbPath, "db", "./database", "database path")

	flag.StringVar(&root, "root", "", "root directory for the http server")
	flag.StringVar(&coverDir, "covers", "covers", "directory for cover images")
	flag.StringVar(&bookDir, "books", "books", "directory for books")

	flag.Parse()

	if root != "" && root[0] != '/' {
		// Fixup root path
		root = "/" + root
	}

	rootFeed = feed{
		Links: []link{
			{Rel: "self", Href: root + "/", Type: "application/atom+xml;profile=opds-catalog;kind=navigation"},
			{Rel: "start", Href: root + "/", Type: "application/atom+xml;profile=opds-catalog;kind=navigation"},
		},
		Title:   "uopds",
		Updated: time.Now(),
		Author: author{
			Name: "uopds",
		},
		Entries: []entry{},
	}

	// read database
	var err error
	db, err = openDatabase(dbPath)
	if err != nil {
		panic(err)
	}

	entries, err := db.entries(context.Background())
	if err != nil {
		panic(err)
	}

	// generate entries in new catalog
	dir, err := os.ReadDir(bookDir)
	if err != nil {
		panic(err)
	}

	for _, file := range dir {
		if !file.Type().IsRegular() {
			// ignore it
			continue
		}

		// check if it's in the database
		if _, err := db.path(context.Background(), file.Name()); err == nil {
			// it exists in the database
			continue
		}

		switch filepath.Ext(file.Name()) {
		case ".cbz":
			entry, err := importCbz(file.Name())
			if err != nil {
				log.Printf("failed to import %s: %v", file.Name(), err)
				continue
			}

			entries = append(entries, entry)
		case ".epub":
			entry, err := importEpub(file.Name())
			if err != nil {
				log.Printf("failed to import %s: %v", file.Name(), err)
				continue
			}

			entries = append(entries, entry)
		default:
			// unsupported
			continue
		}
	}

	rootFeed.Entries = entries

	_loc := root
	if _loc == "" {
		_loc = "/"
	}

	http.HandleFunc(_loc, func(w http.ResponseWriter, r *http.Request) {
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

	log.Fatal(http.ListenAndServe(addr, nil))
}
