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
	addr, root, coverDir, bookDir, dbPath string
)

var rootFeed = feed{
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

func main() {
	flag.StringVar(&addr, "addr", ":8080", "listen address")
	flag.StringVar(&dbPath, "db", "./database", "database path")

	flag.StringVar(&root, "root", "", "root directory for the http server")
	flag.StringVar(&coverDir, "covers", "covers", "directory for cover images")
	flag.StringVar(&bookDir, "books", "books", "directory for books")

	flag.Parse()

	if root == "" || root[0] != '/' {
		// Fixup root path
		root = "/" + root
	}

	// read database
	db, err := openDatabase(dbPath)
	if err != nil {
		panic(err)
	}

	// generate entries in new catalog
	dir, err := os.ReadDir(bookDir)
	if err != nil {
		panic(err)
	}

	dbChanged := false

	for _, file := range dir {
		if !file.Type().IsRegular() || filepath.Ext(file.Name()) != ".epub" {
			// ignore it
			continue
		}

		// check if it's in the database
		var entry entry
		ok := false

		for _, row := range db.rows {
			if row.path == file.Name() {
				// yes it is!
				entry = row.entry()
				ok = true
			}
		}

		if !ok {
			// generate an entry for it
			opf, err := readOpfFromEpub(file.Name())
			if err != nil {
				log.Printf("failed to read %s: %v", file.Name(), err)
				continue
			}

			entry, err = opf.genEntry()
			if err != nil {
				panic(err)
			}

			// add it to the database
			dbChanged = true
			db.add(entry)
		}
	}

	if dbChanged {
		if err := db.commit(); err != nil {
			panic(err)
		}
	}

	// write to rootFeed
	for _, row := range db.rows {
		rootFeed.Entries = append(rootFeed.Entries, row.entry())
	}

	http.HandleFunc(root, func(w http.ResponseWriter, r *http.Request) {
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
