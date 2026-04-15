package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"time"
)

type Feed struct {
	XMLName   xml.Name   `xml:"http://www.w3.org/2005/Atom feed"`
	Generator *Generator `xml:"generator,omitempty"` // XXX: pointer to play nice with omitempty
	Title     string     `xml:"title"`
	ID        string     `xml:"id"`
	Links     []Link     `xml:"link"`
	Updated   AtomTime   `xml:"updated"`
	Authors   []Person   `xml:"author"`
	Entries   []Entry    `xml:"entry"`
}

type Entry struct {
	Title      string     `xml:"title"`
	ID         string     `xml:"id"`
	Links      []Link     `xml:"link"`
	Published  AtomTime   `xml:"published"`
	Updated    AtomTime   `xml:"updated"`
	Authors    []Person   `xml:"author"`
	Categories []Category `xml:"category,omitempty"`
	Summary    string     `xml:"summary,omitempty"`
	Content    *Content   `xml:"content,omitempty"` // XXX: pointer to play nice with omitempty
}

type Generator struct {
	URI     string `xml:"uri,attr,omitempty"`
	Version string `xml:"version,attr,omitempty"`
	Name    string `xml:",chardata"`
}

type Link struct {
	Rel  string `xml:"rel,attr,omitempty"`
	Type string `xml:"type,attr,omitempty"`
	Href string `xml:"href,attr"`
}

type Person struct {
	Name  string `xml:"name"`
	URI   string `xml:"uri,omitempty"`
	Email string `xml:"email,omitempty"`
}

type Category struct {
	Term   string `xml:"term,attr"`
	Scheme string `xml:"scheme,attr,omitempty"`
	Label  string `xml:"label,attr,omitempty"`
}

type Content struct {
	Type string `xml:"type,attr"`
	Body string `xml:",chardata"`
}

type AtomTime time.Time

func (at *AtomTime) Compare(u AtomTime) int {
	return time.Time(*at).Compare(time.Time(u))
}

func (at *AtomTime) Format(layout string) string {
	return time.Time(*at).Format(layout)
}

func (at *AtomTime) MarshalText() ([]byte, error) {
	return []byte(at.Format(time.RFC3339)), nil
}

func (feed *Feed) WriteXML(w io.Writer) error {
	b, err := xml.MarshalIndent(feed, "", "  ")
	if err != nil {
		return err
	}
	w.Write([]byte(xml.Header))
	w.Write(b)
	w.Write([]byte{'\n'})
	return nil
}

func TagURI(tagger string, specific string) string {
	return fmt.Sprintf("tag:%s:%s", tagger, specific)
}
