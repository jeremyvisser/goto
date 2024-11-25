package main

import (
	"encoding/xml"
	"log"
	"net/http"
	"net/url"
)

func ServeOpenSearchXML(w http.ResponseWriter, r *http.Request) {
	if *baseURL == "" {
		http.NotFound(w, r)
		return
	}
	bu, err := url.Parse(*baseURL)
	if err != nil {
		log.Printf("baseURL: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	type Url struct {
		Type     string `xml:"type,attr"`
		Rel      string `xml:"rel,attr,omitempty"`
		Template string `xml:"template,attr"`
	}
	type OpenSearchDescription struct {
		XMLName       any `xml:"http://a9.com/-/spec/opensearch/1.1/ OpenSearchDescription"`
		ShortName     string
		Description   string
		Tags          string
		InputEncoding string
		Url           []Url
	}
	osd := OpenSearchDescription{
		nil,
		"Goto",
		"Link forwarder",
		"goto go",
		"UTF-8",
		[]Url{
			{
				Type:     "text/html",
				Template: bu.String() + "/{searchTerms}", // avoid encoding "{" and "}"
			},
			{
				Type:     "application/opensearchdescription+xml",
				Rel:      "self",
				Template: bu.JoinPath("opensearch.xml").String(),
			},
		},
	}
	w.Header().Add("Content-Type", "application/opensearchdescription+xml")
	w.Write([]byte(xml.Header))
	enc := xml.NewEncoder(w)
	enc.Indent("", "\t")
	if err := enc.Encode(&osd); err != nil {
		log.Print(err)
		http.Error(w, "xml error", http.StatusInternalServerError)
	}
}
