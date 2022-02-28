package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/base32"
	"encoding/hex"
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
	err = db.add(context.Background(), entry, path, "", opf.CoverType)
	return entry, err
}

func importCbz(path string) (entry, error) {
	// There is no metadata standard for CBZ files.
	// Until there is one, and it is widely supported, we will just use the
	// filename as the title and Unknown as the author.
	// You can set it manually in the database if you wish.
	//
	// Most fields in the OPDS feed will be left blank.

	title := strings.TrimSuffix(path, ".cbz")
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

	stat, err := fp.Stat()
	if err != nil {
		return entry, err
	}

	zr, err := zip.NewReader(fp, stat.Size())
	if err != nil {
		return entry, err
	}

	// Fetch the *first* image.
	if len(zr.File) == 0 {
		// Empty???
		return entry, fmt.Errorf("cbz is empty")
	}

	first := zr.File[0]
	firstfp, err := first.Open()
	if err != nil {
		return entry, err
	}
	defer firstfp.Close()

	// Hash it
	digest := sha1.New()
	buf := &bytes.Buffer{}
	tee := io.TeeReader(firstfp, buf)
	if _, err := io.Copy(digest, tee); err != nil {
		return entry, err
	}

	// TODO: It is dangerous to assume it's a jpeg.
	fname := hex.EncodeToString(digest.Sum(nil)) + ".jpg"

	// Can't seek, gotta close and reopen it...
	firstfp.Close()
	firstfp, err = first.Open()
	if err != nil {
		return entry, err
	}

	// Write it out!
	if err := os.WriteFile(coverDir+"/"+fname, buf.Bytes(), 0o644); err != nil {
		return entry, err
	}

	// generate urn
	digest = sha1.New()

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
	err = db.add(context.Background(), entry, path, fname, "image/jpeg")
	return entry, err
}
