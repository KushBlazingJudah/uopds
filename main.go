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
	addr, root, bookDir, dbPath string
	db                          *database
)

func genFeed(rpath string) (feed, error) {
	// Clean the file path to prevent path traversal attacks
	rpath = filepath.Clean(rpath)
	lpath := filepath.Join(bookDir, rpath)

	// Base feed
	f := feed{
		Links: []link{
			{Rel: "self", Href: root + "/" + rpath, Type: opdsAcquisition},
			{Rel: "start", Href: root + "/", Type: opdsAcquisition},
		},
		Title:   rpath,
		Updated: time.Now(),
		Author: author{
			Name: "uopds",
		},
		Entries: []entry{},
	}

	// Don't add an "up" entry if this is the root folder
	if up := filepath.Dir(rpath); up != "." {
		f.Links = append(f.Links, link{Rel: "up", Href: root + "/" + up, Type: opdsAcquisition})
	}

	// Generate entries from folder
	files, err := os.ReadDir(lpath)
	if err != nil {
		return f, err
	}

	for _, file := range files {
		relPath := filepath.Join(rpath, file.Name())

		if file.IsDir() {
			// it's a directory, add an entry for it
			f.Entries = append(f.Entries, entry{
				Title:   file.Name(),
				Author:  author{},
				Links:   []link{{Rel: "subsection", Href: filepath.Join(root, rpath, file.Name()), Type: opdsAcquisition}},
				Updated: time.Now(),
				Content: content{},
			})
			continue
		} else if !file.Type().IsRegular() {
			// ignore it
			continue
		}

		// check extension
		ext := filepath.Ext(file.Name())
		if ext != ".cbz" && ext != ".epub" {
			// something else we don't care about
			continue
		}

		var e entry

		// check if it's in the database
		if e, err = db.path(context.Background(), relPath); err != nil {
			// it is not in the database, new file!
			// index it, and add it to the database.
			switch ext {
			case ".cbz":
				e, err = importCbz(relPath)
			case ".epub":
				e, err = importEpub(relPath)

			default:
				// unsupported
				continue
			}

			if err != nil {
				log.Printf("failed to import %s: %v", relPath, err)
				continue
			}
		}

		f.Entries = append(f.Entries, e)
	}

	return f, nil
}

func main() {
	flag.StringVar(&addr, "addr", ":8080", "listen address")
	flag.StringVar(&dbPath, "db", "./database", "database path")

	flag.StringVar(&root, "root", "", "root directory for the http server")
	flag.StringVar(&bookDir, "books", "books", "directory for books")

	flag.Parse()

	if root != "" && root[0] != '/' {
		// Fixup root path
		root = "/" + root
	}

	// read database
	var err error
	db, err = openDatabase(dbPath)
	if err != nil {
		panic(err)
	}

	_loc := root
	if _loc == "" {
		_loc = "/"
	}

	http.HandleFunc(_loc, func(w http.ResponseWriter, r *http.Request) {
		f, err := genFeed(r.URL.Path)
		if err != nil {
			w.WriteHeader(503)
			fmt.Fprint(w, err)
			return
		}

		out, err := xml.Marshal(f)
		if err != nil {
			w.WriteHeader(503)
			fmt.Fprint(w, err)
			return
		}

		w.Write(out)
	})

	http.Handle(root+"/books/", http.StripPrefix(root+"/books/", http.FileServer(http.Dir(bookDir))))

	log.Fatal(http.ListenAndServe(addr, nil))
}
