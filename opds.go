package main

import (
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

type content struct {
	Type    string `xml:"type,attr"`
	Content string `xml:",innerxml"`
}

type entry struct {
	Title   string    `xml:"title"`
	Author  author    `xml:"author,omitempty"`
	Links   []link    `xml:"link"`
	ID      string    `xml:"id"`
	Updated time.Time `xml:"updated"`

	Summary  string `xml:"summary,omitempty"`
	Language string `xml:"dc:language,omitempty"`
	Date     string `xml:"dc:date,omitempty"`

	Content content `xml:"content,omitempty"`
}

type feed struct {
	Id      string    `xml:"id"`
	Links   []link    `xml:"link"`
	Title   string    `xml:"title"`
	Updated time.Time `xml:"updated"`
	Author  author    `xml:"author"`

	Entries []entry `xml:"entry"`
}
