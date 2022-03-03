package main

import (
	"context"
	"crypto/sha1"
	"encoding/base32"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type importer func(string) (entry, error)

var importers = map[string]importer{
	// File types with support
	".epub": importEpub,

	// Wishlist for support for these types of files.
	// TODO.
	".pdf":  importGeneric,
	".mobi": importGeneric,

	// File types that have no metadata, and will never be supported.
	// These entries are here solely so the entries don't get ditched when
	// reading the directory.
	".cbz": importGeneric,
	".txt": importGeneric,
	"":     importGeneric,
}

// genUrn generates an urn:sha1 ID based on a file.
func genUrn(path string) (string, error) {
	// Open the file for hashing.
	fp, err := os.Open(filepath.Join(bookDir, path))
	if err != nil {
		return "", err
	}
	defer fp.Close()

	// Generate the urn
	digest := sha1.New()
	if _, err := io.Copy(digest, fp); err != nil {
		return "", err
	}

	hash := digest.Sum(nil)

	// urn:sha1 asks that we use base32 without padding.
	enc := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(hash)

	return enc, err
}

// importEpub imports an EPUB into the database by reading the contained metadata.
func importEpub(path string) (entry, error) {
	opf, err := readOpfFromEpub(path)
	if err != nil {
		return entry{}, err
	}

	entry, err := opf.genEntry()
	if err != nil {
		return entry, err
	}

	// generate urn
	urn, err := genUrn(path)
	if err != nil {
		return entry, err
	}

	entry.ID = fmt.Sprintf("urn:sha1:%s", urn)

	// add it to the database
	err = db.add(context.Background(), entry, path)
	return entry, err
}

// importGeneric imports a file into uopds's database without actually pulling anything meaningful from the file.
// This importer is used for files that don't/won't have support in uopds.
func importGeneric(path string) (entry, error) {
	ext := filepath.Ext(path)
	mime := mime.TypeByExtension(ext)

	// mime may be empty, which means it couldn't find a suitable mimetype.
	// Default to application/octet-stream, a generic mimetype.
	if mime == "" {
		mime = "application/octet-stream"
	}

	entry := entry{
		// Infer the title from the filename; assume the filename sans extension is fine.
		Title: strings.TrimSuffix(filepath.Base(path), ext),

		Author: author{Name: "Unknown author"},
		Links: []link{
			{
				Rel:  "http://opds-spec.org/acquisition",
				Href: filepath.Join(bookDir, path),
				Type: mime,
			},
		},
		Updated: time.Now(),
	}

	// generate urn
	urn, err := genUrn(path)
	if err != nil {
		return entry, err
	}

	entry.ID = fmt.Sprintf("urn:sha1:%s", urn)

	// add it to the database
	err = db.add(context.Background(), entry, path)
	return entry, err
}
