package main

import (
	"encoding/xml"
	"time"
)

const (
	opdsAcquisition = "application/atom+xml;profile=opds-catalog;kind=acquisition"
)

type link struct {
	Rel  string `xml:"rel,attr"`
	Href string `xml:"href,attr"`
	Type string `xml:"type,attr"`
}

type author struct {
	Name string `xml:"name"`
}

type summary string

func (s summary) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "type"}, Value: "text"})
	e.EncodeElement(string(s), start)
	return nil
}

type entry struct {
	Title   string    `xml:"title"`            // necessary
	Author  author    `xml:"author,omitempty"` // optional, but should
	Links   []link    `xml:"link"`
	ID      string    `xml:"id"`      // necessary, should be unique
	Updated time.Time `xml:"updated"` // necessary

	Language string `xml:"dc:language,omitempty"` // optional
	Date     string `xml:"dc:date,omitempty"`     // optional

	Summary summary `xml:"content,omitempty"` // optional
}

type feed struct {
	entry

	Entries []entry `xml:"entry"`
}
