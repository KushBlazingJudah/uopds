package main

import (
	"archive/zip"
	"encoding/xml"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/net/html/charset"
)

type metaInf struct {
	Rootfile struct {
		Path string `xml:"full-path,attr"`
	} `xml:"rootfiles>rootfile"`
}

type opfPackage struct {
	Title       string `xml:"metadata>title"`
	Date        string `xml:"metadata>date"`
	Creator     string `xml:"metadata>creator"`
	Description string `xml:"metadata>description"`
	file        string
}

func readXMLZip(path string, zr *zip.Reader, v interface{}) error {
	opf, err := zr.Open(path)
	if err != nil {
		return err
	}
	defer opf.Close()

	decoder := xml.NewDecoder(opf)

	// Ensure files in the incorrect format (i.e. not UTF-8) still get read
	decoder.CharsetReader = charset.NewReaderLabel

	return decoder.Decode(v)
}

func (pkg opfPackage) genEntry() (entry, error) {
	/* if err := os.Mkdir(coverDir, 0); err != nil {
		return entry{}, err
	} */

	e := entry{
		Title:  pkg.Title,
		Author: author{Name: pkg.Creator},
		Links: []link{
			{
				Rel:  "http://opds-spec.org/acquisition",
				Href: filepath.Join(root, pkg.file),
				Type: "application/epub+zip",
			},
		},
		Updated: time.Now(),
		Summary: summary(pkg.Description),
		Date:    pkg.Date,
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

	pkg := opfPackage{file: file}

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
