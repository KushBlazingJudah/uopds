package main

import (
	"archive/zip"
	"encoding/xml"
	"io/ioutil"
)

func readXMLZip(path string, zr *zip.Reader, v interface{}) error {
	opf, err := zr.Open(path)
	if err != nil {
		return err
	}
	defer opf.Close()

	data, err := ioutil.ReadAll(opf)
	if err != nil {
		return err
	}

	err = xml.Unmarshal(data, v)
	return err
}
