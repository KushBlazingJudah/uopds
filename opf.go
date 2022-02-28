package main

import (
	"archive/zip"
	"crypto/sha1"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"mime"
	"os"
	"syscall"
	"time"
)

type opfIdentifier struct {
	Scheme string `xml:"opf:scheme,attr"`
	Value  string `xml:",innerxml"`
}

type opfMetadata struct {
	Title       string    `xml:"title"`
	Language    string    `xml:"language"`
	Date        time.Time `xml:"date"`
	Identifier  string    `xml:"identifier"`
	Creator     string    `xml:"creator"`
	Description string    `xml:"description"`
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
	Created   time.Time
	Updated   time.Time
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
				Href: pkg.File,
				Type: "application/epub+zip",
			},
		},
		ID:       &uuidurn{},
		Updated:  pkg.Updated,
		Summary:  "a book",
		Date:     pkg.Metadata.Date,
		Language: pkg.Metadata.Language,
		Content:  content{Type: "text", Content: pkg.Metadata.Description},
	}

	// hash cover with sha1
	if len(pkg.Cover) > 0 {
		_tmp := sha1.Sum(pkg.Cover)
		hash := hex.EncodeToString(_tmp[:])

		exts, err := mime.ExtensionsByType(pkg.CoverType)
		if err != nil {
			return e, err
		}

		if len(exts) == 0 {
			exts = []string{""}
		}

		fname := fmt.Sprintf("%s%s", hash, exts[0])

		// write out
		if err := os.WriteFile(coverDir+"/"+fname, pkg.Cover, 0o644); err != nil {
			return e, err
		}

		e.Links = append(e.Links, link{
			Rel:  "http://opds-spec.org/image",
			Href: coverDir + "/" + fname,
			Type: "image/" + pkg.CoverType,
		})
	}

	// make entry!
	return e, nil
}

func readOpfFromEpub(file string) (opfPackage, error) {
	epub, err := os.Open(file)
	if err != nil {
		return opfPackage{}, err
	}
	defer epub.Close()

	// get file stats
	stat, err := epub.Stat()
	if err != nil {
		return opfPackage{}, err
	}

	// TODO: change if windows gets explicit support
	stat_t := stat.Sys().(*syscall.Stat_t)

	pkg := opfPackage{
		File: file,

		Created: time.Unix(stat_t.Ctim.Sec, stat_t.Ctim.Nsec),
		Updated: time.Unix(stat_t.Mtim.Sec, stat_t.Mtim.Nsec),
	}

	zr, err := zip.NewReader(epub, stat.Size())
	if err != nil {
		return opfPackage{}, err
	}

	opf, err := zr.Open("content.opf")
	if err != nil {
		return opfPackage{}, err
	}

	data, err := ioutil.ReadAll(opf)
	if err != nil {
		return pkg, err
	}

	// We can close the opf file now because we read it
	opf.Close()

	if err = xml.Unmarshal(data, &pkg); err != nil {
		return pkg, err
	}

	// try to read cover
	for _, i := range pkg.Manifest.Items {
		if i.ID == "cover" {
			fmt.Println(i.Href)

			cover, err := zr.Open(i.Href)
			if err != nil {
				return pkg, err
			}
			defer cover.Close()

			if pkg.Cover, err = ioutil.ReadAll(cover); err != nil {
				return pkg, err
			}

			pkg.CoverType = i.Type

			break
		}
	}

	return pkg, nil
}
