package main

import (
	"encoding/xml"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type urn interface {
	String() string
	MarshalXML(e *xml.Encoder, start xml.StartElement) error
}

type uuidurn struct {
	Value string
}

func (urn *uuidurn) String() string {
	if urn.Value == "" {
		urn.Value = uuid.New().String()
	}

	return fmt.Sprintf("urn:uuid:%s", urn.Value)
}

func (u uuidurn) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return e.EncodeElement(u.String(), start)
}

type link struct {
	Rel  string `xml:"rel,attr"`
	Href string `xml:"href,attr"`
	Type string `xml:"type,attr"`
}

type author struct {
	Name string
	URI  urn
}

type content struct {
	Type    string `xml:"type,attr"`
	Context string `xml:",innerxml"`
}

type entry struct {
	Title   string    `xml:"title"`
	Author  author    `xml:"author,omitempty"`
	Links   []link    `xml:"link"`
	ID      urn       `xml:"id"`
	Updated time.Time `xml:"updated"`

	Summary string `xml:"summary,omitempty"`

	Content content `xml:"content"`
}

type feed struct {
	Id      urn       `xml:"id"`
	Links   []link    `xml:"link"`
	Title   string    `xml:"title"`
	Updated time.Time `xml:"updated"`
	Author  author

	Entries []entry `xml:"entry"`
}
