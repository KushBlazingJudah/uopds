package main

import (
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const root = ""
const staticDir = "static"
const coverDir = staticDir + "/covers"
const bookDir = staticDir + "/books"

func main() {
	testFeed := feed{
		Id: &uuidurn{},
		Links: []link{
			link{Rel: "self", Href: root + "/root.xml", Type: "application/atom+xml;profile=opds-catalog;kind=navigation"},
			link{Rel: "start", Href: root + "/root.xml", Type: "application/atom+xml;profile=opds-catalog;kind=navigation"},
		},
		Title:   "uopds",
		Updated: time.Now(),
		Author: author{
			Name: "uopds",
			URI:  &uuidurn{},
		},
		Entries: []entry{
			entry{
				Title: "here",
				Links: []link{{
					Rel:  "http://opds-spec.org/sort/new",
					Href: "/root.xml",
					Type: "application/atom+xml;profile=opds-catalog;kind=acquisition",
				}},
				Updated: time.Now(),
				ID:      &uuidurn{},
				Content: content{Type: "text", Content: "wow"},
			},
			entry{
				Title: "test entry",
				Links: []link{{
					Rel:  "http://opds-spec.org/sort/new",
					Href: "/new.xml",
					Type: "application/atom+xml;profile=opds-catalog;kind=acquisition",
				}},
				Updated: time.Now(),
				ID:      &uuidurn{},
				Content: content{Type: "text", Content: "cool"},
			},
		},
	}

	newFeed := feed{
		Id: &uuidurn{},
		Links: []link{
			link{Rel: "self", Href: root + "/new.xml", Type: "application/atom+xml;profile=opds-catalog;kind=navigation"},
			link{Rel: "start", Href: root + "/root.xml", Type: "application/atom+xml;profile=opds-catalog;kind=navigation"},
			link{Rel: "up", Href: root + "/root.xml", Type: "application/atom+xml;profile=opds-catalog;kind=navigation"},
		},

		Title:   "New files",
		Updated: time.Now(),
		Author: author{
			Name: "uopds",
			URI:  &uuidurn{},
		},

		Entries: []entry{
			entry{
				Title: "my cool book",
				Links: []link{
					{
						Rel:  "http://opds-spec.org/thumbnail",
						Href: "/static/test.png",
						Type: "image/png",
					},
					{
						Rel:  "http://opds-spec.org/image",
						Href: "/static/test.png",
						Type: "image/png",
					},
					{
						Rel:  "http://opds-spec.org/acquisition",
						Href: "/static/test.epub",
						Type: "application/epub+zip",
					},
				},
				Updated: time.Now(),
				ID:      &uuidurn{},
				Content: content{Type: "text", Content: "this book is really cool"},
				Author: author{
					Name: "cool book writer",
					URI:  &uuidurn{},
				},
			},
		},
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
		opf, err := readOpfFromEpub(filepath.Join(bookDir, file.Name()))
		if err != nil {
			panic(err)
		}

		e, err := opf.genEntry()
		if err != nil {
			panic(err)
		}

		newFeed.Entries = append(newFeed.Entries, e)
	}

	http.HandleFunc(root+"/root.xml", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s: GET root", r.RemoteAddr)

		out, err := xml.Marshal(testFeed)
		if err != nil {
			w.WriteHeader(503)
			fmt.Fprint(w, err)
			return
		}

		w.Write(out)
	})

	http.HandleFunc(root+"/new.xml", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s: GET new", r.RemoteAddr)

		out, err := xml.Marshal(newFeed)
		if err != nil {
			w.WriteHeader(503)
			fmt.Fprint(w, err)
			return
		}

		w.Write(out)
	})

	http.Handle(root+"/static/", http.StripPrefix(root+"/static/", http.FileServer(http.Dir("./static"))))

	log.Fatal(http.ListenAndServe(":8080", nil))
}
