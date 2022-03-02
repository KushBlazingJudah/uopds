package main

import (
	"archive/zip"
	"os"
	"path/filepath"
	"time"
)

type metaInf struct {
	Rootfile struct {
		Path string `xml:"full-path,attr"`
	} `xml:"rootfiles>rootfile"`
}

type opfIdentifier struct {
	Scheme string `xml:"opf:scheme,attr"`
	Value  string `xml:",innerxml"`
}

type opfMetadata struct {
	Title       string        `xml:"title"`
	Language    string        `xml:"language"`
	Date        string        `xml:"date"`
	Identifier  opfIdentifier `xml:"identifier"`
	Creator     string        `xml:"creator"`
	Description string        `xml:"description"`
}

type opfItem struct {
	Href string `xml:"href,attr"`
	ID   string `xml:"id,attr"`
	Type string `xml:"media-type,attr"`
}

type opfManifest struct {
	Items []opfItem `xml:"item"`
}

type opfPackage struct {
	Metadata  opfMetadata `xml:"metadata"`
	Manifest  opfManifest `xml:"manifest"`
	Cover     []byte
	CoverType string
	File      string
}

func (pkg opfPackage) genEntry() (entry, error) {
	/* if err := os.Mkdir(coverDir, 0); err != nil {
		return entry{}, err
	} */

	e := entry{
		Title:  pkg.Metadata.Title,
		Author: author{Name: pkg.Metadata.Creator},
		Links: []link{
			{
				Rel:  "http://opds-spec.org/acquisition",
				Href: root + "/books/" + pkg.File,
				Type: "application/epub+zip",
			},
		},
		Updated:  time.Now(),
		Summary:  pkg.Metadata.Description,
		Date:     pkg.Metadata.Date,
		Language: pkg.Metadata.Language,
	}

	// make entry!
	return e, nil
}

func readOpfFromEpub(file string) (opfPackage, error) {
	epub, err := os.Open(filepath.Join(bookDir, file))
	if err != nil {
		return opfPackage{}, err
	}
	defer epub.Close()

	// get file stats
	stat, err := epub.Stat()
	if err != nil {
		return opfPackage{}, err
	}

	pkg := opfPackage{File: file}

	zr, err := zip.NewReader(epub, stat.Size())
	if err != nil {
		return pkg, err
	}

	// read META-INF
	var meta metaInf
	if err := readXMLZip("META-INF/container.xml", zr, &meta); err != nil {
		return pkg, err
	}

	// try to read metadata
	if err := readXMLZip(meta.Rootfile.Path, zr, &pkg); err != nil {
		return pkg, err
	}

	return pkg, nil
}
