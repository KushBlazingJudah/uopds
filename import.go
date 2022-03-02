package main

import (
	"context"
	"crypto/sha1"
	"encoding/base32"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func importEpub(path string) (entry, error) {
	// generate an entry for it
	opf, err := readOpfFromEpub(path)
	if err != nil {
		return entry{}, err
	}

	entry, err := opf.genEntry()
	if err != nil {
		return entry, err
	}

	// generate urn
	digest := sha1.New()

	f, err := os.Open(filepath.Join(bookDir, path))
	if err != nil {
		panic(err)
	}

	if _, err := io.Copy(digest, f); err != nil {
		panic(err)
	}

	f.Close()

	hash := digest.Sum(nil)
	enc := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(hash)

	entry.ID = fmt.Sprintf("urn:sha1:%s", enc)

	// add it to the database
	err = db.add(context.Background(), entry, path)
	return entry, err
}

func importCbz(path string) (entry, error) {
	// There is no metadata standard for CBZ files.
	// Until there is one, and it is widely supported, we will just use the
	// filename as the title and Unknown as the author.
	// You can set it manually in the database if you wish.
	//
	// Most fields in the OPDS feed will be left blank.

	title := strings.TrimSuffix(filepath.Base(path), ".cbz")
	entry := entry{
		Title:  title,
		Author: author{Name: "Unknown author"},
		Links: []link{
			{
				Rel:  "http://opds-spec.org/acquisition",
				Href: root + "/books/" + path,
				Type: "application/x-cbz",
			},
		},
		Updated: time.Now(),
	}

	// The cover image is the first image.
	// Get it.
	fp, err := os.Open(filepath.Join(bookDir, path))
	if err != nil {
		return entry, err
	}
	defer fp.Close()

	// generate urn
	digest := sha1.New()

	if _, err := io.Copy(digest, fp); err != nil {
		panic(err)
	}

	hash := digest.Sum(nil)
	enc := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(hash)

	entry.ID = fmt.Sprintf("urn:sha1:%s", enc)

	// add it to the database
	err = db.add(context.Background(), entry, path)
	return entry, err
}
