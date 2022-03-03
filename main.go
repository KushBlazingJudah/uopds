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
	f := feed{
		Links: []link{
			{Rel: "self", Href: filepath.Join(root, "/", rpath), Type: opdsAcquisition},
			{Rel: "start", Href: filepath.Join(root, "/"), Type: opdsAcquisition},
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
		// Setup a redirect
		http.Handle("/", http.RedirectHandler(root, http.StatusMovedPermanently))
	}

	http.Handle(_loc, http.StripPrefix(_loc, opds{}))

	log.Fatal(http.ListenAndServe(addr, nil))
}
