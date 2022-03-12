package main

import (
	"database/sql"
	"errors"
	"net/url"
	"path/filepath"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

const sqlSchema = `
CREATE TABLE IF NOT EXISTS books(
	id INTEGER NOT NULL PRIMARY KEY,

	path TEXT NOT NULL,
	urn TEXT NOT NULL,

	title TEXT,
	author TEXT,
	language TEXT,
	summary TEXT,

	date TEXT,

	UNIQUE(path)
);

CREATE TABLE IF NOT EXISTS dirs(
	path TEXT NOT NULL,
	urn TEXT NOT NULL,

	UNIQUE(path),
	UNIQUE(urn)
);
`

type database struct {
	conn *sql.DB
}

func openDatabase(path string) (*database, error) {
	conn, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	// run table schema
	_, err = conn.Exec(sqlSchema)
	if err != nil {
		conn.Close()
		return nil, err
	}

	return &database{conn: conn}, nil
}

func (db *database) path(path string) (entry, error) {
	var (
		source string
		e      entry
	)

	row := db.conn.QueryRow("SELECT path, urn, title, author, language, summary, date FROM books WHERE path = ?", path)
	err := row.Scan(&source, &e.ID, &e.Title, &e.Author.Name, &e.Language, &e.Summary, &e.Date)

	e.Links = []link{
		{
			Rel: "http://opds-spec.org/acquisition",

			// We don't ever perform anything with the entry that is returned
			// by this function so add on the root path to the source.
			// It makes life mildly easier.
			Href: url.URL{Path: filepath.Join(root, source)},

			Type: "application/epub+zip",
		},
	}

	// Get the modify time for the file
	e.Updated = modTime(path)

	return e, err
}

func (db *database) add(e entry, source string) error {
	named := []interface{}{
		sql.Named("path", source),
		sql.Named("urn", e.ID),
		sql.Named("title", e.Title),
		sql.Named("author", e.Author.Name),
		sql.Named("language", e.Language),
		sql.Named("summary", e.Summary),
		sql.Named("date", e.Date),
	}

	_, err := db.conn.Exec(`INSERT INTO books(path, urn, title,
author, language, summary, date) VALUES (:path, :urn, :title, :author,
:language, :summary, :date)`, named...)
	return err
}

func (db *database) dir(path string) (string, error) {
	path = filepath.Clean(path)

	r := db.conn.QueryRow(`SELECT urn FROM dirs WHERE path = ?`, path)

	urn := ""
	err := r.Scan(&urn)

	if urn == "" && errors.Is(err, sql.ErrNoRows) {
		// Need to generate one
		urn = uuid.New().URN()

		_, err = db.conn.Exec("INSERT INTO dirs(path, urn) VALUES(?,?)", path, urn)
	}

	return urn, err
}
