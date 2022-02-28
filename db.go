package main

import (
	"context"
	"database/sql"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

const timeFmt = "2006-01-02"

const sqlSchema = `
CREATE TABLE IF NOT EXISTS books(
	id INTEGER NOT NULL PRIMARY KEY,

	path TEXT NOT NULL,
	hash BLOB NOT NULL,
	cover TEXT,
	coverType TEXT,

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

func (db *database) entry(ctx context.Context, id int64) (entry, error) {
	var (
		hash, source, cover, coverType string
		e                              entry
	)

	row := db.conn.QueryRowContext(ctx, "SELECT path, hash, cover, coverType, title, author, language, summary, date FROM books WHERE id = ?", id)
	if err := row.Scan(&source, &hash, &cover, &coverType, &e.Title, &e.Author, &e.Language, &e.Language, &e.Summary, &e.Date); err != nil {
		return e, err
	}

	e.Links = []link{
		{
			Rel:  "http://opds-spec.org/acquisition",
			Href: root + "/books/" + source,
			Type: "application/epub+zip",
		},
	}

	e.Content = content{Type: "text", Content: e.Summary}

	// add cover if it exists
	if len(cover) > 0 {
		e.Links = append(e.Links, link{
			Rel:  "http://opds-spec.org/image",
			Href: root + "/covers/" + cover,
			Type: coverType,
		})
	}

	return e, nil
}

func (db *database) path(ctx context.Context, path string) (entry, error) {
	var (
		hash, source, cover, coverType string
		e                              entry
	)

	row := db.conn.QueryRowContext(ctx, "SELECT path, hash, cover, coverType, title, author, language, summary, date FROM books WHERE path = ?", path)
	if err := row.Scan(&source, &hash, &cover, &coverType, &e.Title, &e.Author, &e.Language, &e.Language, &e.Summary, &e.Date); err != nil {
		return e, err
	}

	e.Links = []link{
		{
			Rel:  "http://opds-spec.org/acquisition",
			Href: root + "/books/" + source,
			Type: "application/epub+zip",
		},
	}

	e.Content = content{Type: "text", Content: e.Summary}

	// add cover if it exists
	if len(cover) > 0 {
		e.Links = append(e.Links, link{
			Rel:  "http://opds-spec.org/image",
			Href: root + "/covers/" + cover,
			Type: coverType,
		})
	}

	return e, nil
}

func (db *database) entries(ctx context.Context) ([]entry, error) {
	rows, err := db.conn.QueryContext(ctx, "SELECT path, hash, cover, coverType, title, author, language, summary, date FROM books")
	if err != nil {
		return nil, err
	}

	entries := []entry{}

	for rows.Next() {
		var (
			hash, source, cover, coverType string
			e                              entry
		)

		if err := rows.Scan(&source, &hash, &cover, &coverType, &e.Title, &e.Author, &e.Language, &e.Language, &e.Summary, &e.Date); err != nil {
			return entries, err
		}

		e.Links = []link{
			{
				Rel:  "http://opds-spec.org/acquisition",
				Href: root + "/books/" + source,
				Type: "application/epub+zip",
			},
		}

		e.Content = content{Type: "text", Content: e.Summary}

		// add cover if it exists
		if len(cover) > 0 {
			e.Links = append(e.Links, link{
				Rel:  "http://opds-spec.org/image",
				Href: root + "/covers/" + cover,
				Type: coverType,
			})
		}

		entries = append(entries, e)
	}

	return entries, nil
}

func (db *database) add(ctx context.Context, e entry, source, cover, coverType, hash string) error {
	named := []interface{}{
		sql.Named("path", source),
		sql.Named("hash", hash),
		sql.Named("cover", cover),
		sql.Named("coverType", coverType),
		sql.Named("title", e.Title),
		sql.Named("author", e.Author.Name),
		sql.Named("language", e.Language),
		sql.Named("summary", e.Summary),
		sql.Named("date", e.Date),
	}

	_, err := db.conn.ExecContext(ctx, `INSERT INTO books(path, hash, cover,
	coverType, title, author, language, summary, date) VALUES (:path, :hash,
	:cover, :coverType, :title, :author, :language, :summary, :date)`, named...)
	return err
}
