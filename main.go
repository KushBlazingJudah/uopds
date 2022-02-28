package main

import (
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"time"
)

const root = ""

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
				Link: link{
					Rel:  "http://opds-spec.org/sort/new",
					Href: "/root.xml",
					Type: "application/atom+xml;profile=opds-catalog;kind=acquisition",
				},
				Updated: time.Now(),
				ID:      &uuidurn{},
				Content: content{Type: "text", Context: "wow"},
			},
			entry{
				Title: "test entry",
				Link: link{
					Rel:  "http://opds-spec.org/sort/new",
					Href: "/new.xml",
					Type: "application/atom+xml;profile=opds-catalog;kind=acquisition",
				},
				Updated: time.Now(),
				ID:      &uuidurn{},
				Content: content{Type: "text", Context: "cool"},
			},
		},
	}

	http.HandleFunc("/root.xml", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s: GET root", r.RemoteAddr)

		out, err := xml.Marshal(testFeed)
		if err != nil {
			w.WriteHeader(503)
			fmt.Fprint(w, err)
			return
		}

		w.Write(out)
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
