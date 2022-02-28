package main

import (
	"archive/zip"
	"encoding/xml"

	"golang.org/x/net/html/charset"
)

func readXMLZip(path string, zr *zip.Reader, v interface{}) error {
	opf, err := zr.Open(path)
	if err != nil {
		return err
	}
	defer opf.Close()

	decoder := xml.NewDecoder(opf)
	decoder.CharsetReader = charset.NewReaderLabel

	err = decoder.Decode(v)
	return err
}
