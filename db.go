package main

import (
	"context"
	"database/sql"
	"path/filepath"
	"sync"
	"time"

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
`

type database struct {
	conn *sql.DB
	mut  sync.Mutex
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

func (db *database) path(ctx context.Context, path string) (entry, error) {
	db.mut.Lock()
	defer db.mut.Unlock()

	var (
		source string
		e      entry
	)

	row := db.conn.QueryRowContext(ctx, "SELECT path, urn, title, author, language, summary, date FROM books WHERE path = ?", path)
	if err := row.Scan(&source, &e.ID, &e.Title, &e.Author.Name, &e.Language, &e.Summary, &e.Date); err != nil {
		return e, err
	}

	e.Links = []link{
		{
			Rel: "http://opds-spec.org/acquisition",

			// We don't ever perform anything with the entry that is returned
			// by this function so add on the root path to the source.
			// It makes life mildly easier.
			Href: filepath.Join(root, source),

			Type: "application/epub+zip",
		},
	}

	// TODO: this is probably not the best thing to do, but i'm told i need it
	e.Updated = time.Now()

	return e, nil
}

func (db *database) add(ctx context.Context, e entry, source string) error {
	db.mut.Lock()
	defer db.mut.Unlock()

	named := []interface{}{
		sql.Named("path", source),
		sql.Named("urn", e.ID),
		sql.Named("title", e.Title),
		sql.Named("author", e.Author.Name),
		sql.Named("language", e.Language),
		sql.Named("summary", e.Summary),
		sql.Named("date", e.Date),
	}

	_, err := db.conn.ExecContext(ctx, `INSERT INTO books(path, urn, title,
author, language, summary, date) VALUES (:path, :urn, :title, :author,
:language, :summary, :date)`, named...)
	return err
}
