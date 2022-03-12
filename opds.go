package main

import (
	"encoding/xml"
	"net/url"
	"time"
)

const opdsAcquisition = "application/atom+xml;profile=opds-catalog;kind=acquisition"

type link struct {
	Rel, Type string
	Href      url.URL
}

func (l link) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Attr = []xml.Attr{
		{Name: xml.Name{Local: "rel"}, Value: l.Rel},
		{Name: xml.Name{Local: "href"}, Value: l.Href.String()},
		{Name: xml.Name{Local: "type"}, Value: l.Type},
	}

	if err := e.EncodeToken(start); err != nil {
		return err
	}

	return e.EncodeToken(xml.EndElement{Name: start.Name})
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
	Author  author    `xml:"author,omitempty"` // optional
	Links   []link    `xml:"link"`
	ID      string    `xml:"id"`      // necessary, should be unique
	Updated time.Time `xml:"updated"` // necessary

	Language string `xml:"dc:language,omitempty"` // optional
	Date     string `xml:"dc:date,omitempty"`     // optional

	Summary summary `xml:"content,omitempty"` // optional
}

type feed struct {
	// Go's native XML encoder has pretty awful support for namespaces, so this is the best way to circumvent it.
	Xmlns   string `xml:"xmlns,attr"`
	XmlnsDc string `xml:"xmlns:dc,attr"`

	entry

	Entries []entry `xml:"entry"`
}
