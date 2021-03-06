package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"time"
)

var (
	addr, root, bookDir, dbPath string
	db                          *database
)

// modTime will attempt to get the modification date for a path.
// If it is unable to, it will return the current date.
func modTime(path string) time.Time {
	stat, err := os.Stat(filepath.Join(bookDir, path))
	if err == nil {
		return stat.ModTime()
	}

	return time.Now()
}

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

		// Send the XML header
		w.Write([]byte(`<?xml version="1.0" encoding="utf-8"?>`))

		out := xml.NewEncoder(w)
		err = out.Encode(f)
		if err != nil {
			log.Printf("error marshalling feed for %s: %v", path, err)
			return
		}
	} else {
		// It is a file, serve it
		http.ServeFile(w, r, lpath)
	}
}

func genFeed(rpath string) (feed, error) {
	// Local path is just the remote path but prefixed with bookDir
	lpath := filepath.Join(bookDir, rpath)

	// Base feed
	f := feed{
		Xmlns:   "http://www.w3.org/2005/Atom",
		XmlnsDc: "http://purl.org/dc/terms/",
	}

	f.Links = []link{
		// Ensure the trailing slash to prevent redirects
		{Rel: "self", Href: url.URL{Path: filepath.Join(root, rpath)}, Type: opdsAcquisition},
		{Rel: "start", Href: url.URL{Path: filepath.Join(root)}, Type: opdsAcquisition},
	}

	f.Title = rpath
	f.Updated = modTime(rpath)

	// Don't add an "up" entry if this is the root folder
	if up := filepath.Dir(rpath); up != "." {
		f.Links = append(f.Links, link{Rel: "up", Href: url.URL{Path: filepath.Join(root, up)}, Type: opdsAcquisition})
	}

	// Generate entries from folder
	dirEntries, err := os.ReadDir(lpath)
	if err != nil {
		return f, err
	}

	dirs := []string{}
	files := []string{}

	for _, file := range dirEntries {
		if file.IsDir() {
			// it's a directory, add an entry for it
			dirs = append(dirs, file.Name())
		} else if file.Type().IsRegular() {
			// it's a file, add an entry for it
			files = append(files, file.Name())
		}
	}

	sort.Strings(dirs)
	sort.Strings(files)

	for _, dir := range dirs {
		// Shouldn't throw away errors, but it isn't vitally important.
		urn, _ := db.dir(filepath.Join(rpath, dir))

		e := entry{
			Title:   dir,
			Links:   []link{{Rel: "subsection", Href: url.URL{Path: filepath.Join(root, rpath, dir)}, Type: opdsAcquisition}},
			Updated: modTime(dir),
			ID:      urn,
		}

		f.Entries = append(f.Entries, e)
	}

	for _, file := range files {
		relPath := filepath.Join(rpath, file)

		var e entry

		// check extension
		ext := filepath.Ext(file)

		// check if it's in the database
		if e, err = db.path(relPath); err != nil {
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

	// Give it a unique ID as required
	f.ID, err = db.dir(lpath)

	return f, err
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
	} else if root == "" {
		root = "/"
	}

	if root[len(root)-1] != '/' {
		// If we don't include the trailing slash, StripPrefix will act in unexpected ways.
		// This has the side effect of making "http://server/root" redirect to "http://server/root/".
		root += "/"
	}

	// read database
	var err error
	db, err = openDatabase(dbPath)
	if err != nil {
		panic(err)
	}

	smux := &http.ServeMux{}

	// Setup a redirect
	if root != "/" {
		smux.Handle("/", http.RedirectHandler(root, http.StatusMovedPermanently))
		smux.Handle(root[:len(root)-1], http.StripPrefix(root[:len(root)-1], opds{}))
	}

	smux.Handle(root, http.StripPrefix(root, opds{}))

	srv := &http.Server{
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		Handler:      smux,
		Addr:         addr,
	}

	log.Fatal(srv.ListenAndServe())
}
