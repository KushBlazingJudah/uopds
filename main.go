package main

import (
	"context"
	"crypto/sha1"
	"encoding/base32"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
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
		if !file.Type().IsRegular() || filepath.Ext(file.Name()) != ".epub" {
			// ignore it
			continue
		}

		// check if it's in the database
		if _, err := db.path(context.Background(), file.Name()); err == nil {
			// it exists in the database
			continue
		}

		// generate an entry for it
		opf, err := readOpfFromEpub(file.Name())
		if err != nil {
			log.Printf("failed to read %s: %v", file.Name(), err)
			continue
		}

		entry, err := opf.genEntry()
		if err != nil {
			panic(err)
		}

		// generate urn
		digest := sha1.New()

		f, err := os.Open(filepath.Join(bookDir, file.Name()))
		if err != nil {
			panic(err)
		}

		if _, err := io.Copy(digest, f); err != nil {
			panic(err)
		}

		f.Close()

		hash := digest.Sum(nil)
		enc := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(hash)
		fmt.Println(enc)
		entry.ID = fmt.Sprintf("urn:sha1:%s", enc)

		// add it to the database
		if err := db.add(context.Background(), entry, file.Name(), "", opf.CoverType); err != nil {
			panic(err)
		}

		entries = append(entries, entry)
	}

	rootFeed.Entries = entries

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
