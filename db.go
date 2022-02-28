package main

import (
	"context"
	"database/sql"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const sqlSchema = `
CREATE TABLE IF NOT EXISTS books(
	id INTEGER NOT NULL PRIMARY KEY,

	path TEXT NOT NULL,
	urn TEXT NOT NULL,
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
	db.mut.Lock()
	defer db.mut.Unlock()

	var (
		source, cover, coverType string
		e                        entry
	)

	row := db.conn.QueryRowContext(ctx, "SELECT path, urn, cover, coverType, title, author, language, summary, date FROM books WHERE id = ?", id)
	if err := row.Scan(&source, &e.ID, &cover, &coverType, &e.Title, &e.Author.Name, &e.Language, &e.Summary, &e.Date); err != nil {
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

	// TODO: this is probably not the best thing to do, but i'm told i need it
	e.Updated = time.Now()

	return e, nil
}

func (db *database) path(ctx context.Context, path string) (entry, error) {
	db.mut.Lock()
	defer db.mut.Unlock()

	var (
		source, cover, coverType string
		e                        entry
	)

	row := db.conn.QueryRowContext(ctx, "SELECT path, urn, cover, coverType, title, author, language, summary, date FROM books WHERE path = ?", path)
	if err := row.Scan(&source, &e.ID, &cover, &coverType, &e.Title, &e.Author.Name, &e.Language, &e.Summary, &e.Date); err != nil {
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

	// TODO: this is probably not the best thing to do, but i'm told i need it
	e.Updated = time.Now()

	return e, nil
}

func (db *database) entries(ctx context.Context) ([]entry, error) {
	db.mut.Lock()
	defer db.mut.Unlock()

	rows, err := db.conn.QueryContext(ctx, "SELECT path, urn, cover, coverType, title, author, language, summary, date FROM books")
	if err != nil {
		return nil, err
	}

	entries := []entry{}

	for rows.Next() {
		var (
			source, cover, coverType string
			e                        entry
		)

		if err := rows.Scan(&source, &e.ID, &cover, &coverType, &e.Title, &e.Author.Name, &e.Language, &e.Summary, &e.Date); err != nil {
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

		// TODO: this is probably not the best thing to do, but i'm told i need it
		e.Updated = time.Now()

		entries = append(entries, e)
	}

	return entries, nil
}

func (db *database) add(ctx context.Context, e entry, source, cover, coverType string) error {
	db.mut.Lock()
	defer db.mut.Unlock()

	named := []interface{}{
		sql.Named("path", source),
		sql.Named("urn", e.ID),
		sql.Named("cover", cover),
		sql.Named("coverType", coverType),
		sql.Named("title", e.Title),
		sql.Named("author", e.Author.Name),
		sql.Named("language", e.Language),
		sql.Named("summary", e.Summary),
		sql.Named("date", e.Date),
	}

	_, err := db.conn.ExecContext(ctx, `INSERT INTO books(path, urn, cover,
	coverType, title, author, language, summary, date) VALUES (:path, :urn,
	:cover, :coverType, :title, :author, :language, :summary, :date)`, named...)
	return err
}
