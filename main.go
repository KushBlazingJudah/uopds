package main

import (
	"context"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
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

// Dummy type for implementing http.Handler
type opds struct{}

func (opds) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Clean path as soon as we can
	path := filepath.Clean(r.URL.Path)

	// Local path is just the remote path but prefixed with bookDir
	lpath := filepath.Join(bookDir, path)

	// Check if it's a directory
	stat, err := os.Stat(lpath)
	if err != nil {
		w.WriteHeader(503)
		fmt.Fprint(w, err)
		return
	}

	if stat.IsDir() {
		// This is a directory, generate a feed for it

		f, err := genFeed(path)
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
		return
	}

	// It is a file, serve it
	fp, err := os.OpenFile(lpath, os.O_RDONLY, 0) // Last arg isn't needed since we won't create it
	if err != nil && errors.Is(err, fs.ErrNotExist) {
		// File doesn't exist, send a 404
		w.WriteHeader(404)
		fmt.Fprint(w, err)
		return
	} else if err != nil {
		// Some other error
		w.WriteHeader(503)
		fmt.Fprint(w, err)
		return
	}

	if _, err := io.Copy(w, fp); err != nil {
		// Close the connection and complain
		log.Printf("error sending file %s: %v", lpath, err)
		return
	}
}

func genFeed(rpath string) (feed, error) {
	// Local path is just the remote path but prefixed with bookDir
	lpath := filepath.Join(bookDir, rpath)

	// Base feed
	f := feed{}

	f.Links = []link{
		// Ensure the trailing slash to prevent redirects
		{Rel: "self", Href: filepath.Join(root, rpath) + "/", Type: opdsAcquisition},
		{Rel: "start", Href: filepath.Join(root) + "/", Type: opdsAcquisition},
	}

	f.Title = rpath
	f.Updated = time.Now()
	f.Author = author{Name: "uopds"}

	// Don't add an "up" entry if this is the root folder
	if up := filepath.Dir(rpath); up != "." {
		f.Links = append(f.Links, link{Rel: "up", Href: filepath.Join(root, up), Type: opdsAcquisition})
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
				Links:   []link{{Rel: "subsection", Href: filepath.Join(root, rpath, file.Name()), Type: opdsAcquisition}},
				Updated: time.Now(),
			})
			continue
		} else if !file.Type().IsRegular() {
			// ignore it
			continue
		}

		var e entry

		// check extension
		ext := filepath.Ext(file.Name())

		// check if it's in the database
		if e, err = db.path(context.Background(), relPath); err != nil {
			// it is not in the database, new file!
			// index it, and add it to the database.
			fn, ok := importers[ext]
			if !ok {
				// This file type is unknown to uopds; if this eats innocuous
				// files, add support to importers in import.go.
				continue
			}

			// There exists an importer for this, try it out.
			if e, err = fn(relPath); err != nil {
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
		// Fixup root path, because it needs a preceeding slash
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
	} else {
		if _loc[len(_loc)-1] != '/' {
			// If we don't include the trailing slash, StripPrefix will act in unexpected ways.
			// This has the side effect of making "http://server/root" redirect to "http://server/root/".
			_loc += "/"
		}

		// Setup a redirect and fix root
		http.Handle("/", http.RedirectHandler(_loc, http.StatusMovedPermanently))
	}

	http.Handle(_loc, http.StripPrefix(_loc, opds{}))

	log.Fatal(http.ListenAndServe(addr, nil))
}
