package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"sync"
	"time"
)

const timeFmt = "2006-01-02"

type row struct {
	path             string
	cover, coverType string

	title, author, language, summary string

	date string
}

type database struct {
	file string
	rows map[int64]row
	mut  sync.Mutex
	i    int64
}

func (r row) entry() entry {
	e := entry{
		Title:  r.title,
		Author: author{Name: r.author},
		Links: []link{
			{
				Rel:  "http://opds-spec.org/acquisition",
				Href: root + "/books/" + r.path,
				Type: "application/epub+zip",
			},
		},
		ID:       &uuidurn{},
		Updated:  time.Now(),
		Summary:  "a book",
		Date:     r.date,
		Language: r.language,
		Content:  content{Type: "text", Content: r.summary},
	}

	// add cover if it exists
	if len(r.cover) > 0 {
		e.Links = append(e.Links, link{
			Rel:  "http://opds-spec.org/image",
			Href: root + "/covers/" + r.cover,
			Type: r.coverType,
		})
	}

	return e
}

func (db *database) add(entry entry) {
	db.mut.Lock()
	defer db.mut.Unlock()

	i := db.i
	db.i++

	r := row{
		path:      entry.sourceFile,
		cover:     entry.coverFile,
		coverType: entry.coverType,
		title:     entry.Title,
		author:    entry.Author.Name,
		language:  entry.Language,
		summary:   entry.Summary,
		date:      entry.Date,
	}

	db.rows[i] = r
}

func (db *database) commit() error {
	file, err := os.OpenFile(db.file, os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	csvw := csv.NewWriter(file)

	db.mut.Lock()
	defer db.mut.Unlock()

	for key, row := range db.rows {
		out := []string{
			fmt.Sprint(key),
			fmt.Sprint(row.path), fmt.Sprint(row.cover), fmt.Sprint(row.coverType),
			fmt.Sprint(row.title), fmt.Sprint(row.author),
			fmt.Sprint(row.language), fmt.Sprint(row.summary),
			timeFmt,
		}

		if err := csvw.Write(out); err != nil {
			return err
		}
	}

	csvw.Flush()
	return csvw.Error()
}

func openDatabase(file string) (*database, error) {
	fp, err := os.OpenFile(file, os.O_RDONLY|os.O_CREATE, 0o644)
	if err != nil {
		return nil, err
	}

	csvr := csv.NewReader(fp)

	db := new(database)
	db.file = file
	db.rows = map[int64]row{}

	rows, err := csvr.ReadAll()
	if err != nil {
		panic(err)
	}

	for _, cr := range rows {
		id, err := strconv.ParseInt(cr[0], 10, 64)
		if err != nil {
			return nil, err
		}

		if id > db.i {
			db.i = id
		}

		r := row{
			path: cr[1], cover: cr[2], coverType: cr[3],
			title: cr[4], author: cr[5],
			language: cr[6], summary: cr[7],
			date: cr[8],
		}

		db.rows[id] = r
	}

	if errors.Is(err, io.EOF) {
		// There was nothing left to read
		err = nil
	}

	return db, err
}
