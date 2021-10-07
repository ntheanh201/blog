package main

import (
	"bytes"
	"fmt"
	"go/format"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/miku/zek"
)

type xmlToGoDownloadResponse struct {
	XML   string `json:"xml,omitempty"`
	Error string `json:"error,omitempty"`
}

// /xmltogo/dlxml
// url: url to download xml from
func handleXMLToGoDownloadXML(w http.ResponseWriter, r *http.Request) {
	var rsp xmlToGoDownloadResponse
	uri := r.FormValue("url")
	if uri == "" {
		rsp.Error = "empty or missing url"
		httpOkWithJSONCompact(w, r, rsp)
		return
	}
	d, err := httpGet(uri)
	if err != nil {
		rsp.Error = err.Error()
		httpOkWithJSONCompact(w, r, rsp)
		return
	}
	rsp.XML = string(d)
	httpOkWithJSONCompact(w, r, rsp)
}

type xmlToGoResponse struct {
	Go    string `json:"go,omitempty"`
	Error string `json:"error,omitempty"`
}

// /xmltogo/convert
// xml: xml to convert
func handleXMLToGoConvert(w http.ResponseWriter, r *http.Request) {
	//fmt.Printf("handleXMLToGoConvert\n")
	var rsp xmlToGoResponse
	xml := r.FormValue("xml")
	if xml == "" {
		rsp.Error = "empty or missing xml"
		httpOkWithJSONCompact(w, r, rsp)
		return
	}

	root := &zek.Node{}
	buf := bytes.NewReader([]byte(xml))
	_, err := root.ReadFrom(buf)
	if err != nil {
		rsp.Error = fmt.Sprintf("root.ReadFrom failed with %s", err)
		httpOkWithJSONCompact(w, r, rsp)
		return
	}
	var res bytes.Buffer
	sw := zek.NewStructWriter(&res)
	//sw.WithComments = true
	//sw.WithJSONTags = true
	//sw.Strict = true
	//sw.ExampleMaxChars = 10
	sw.Compact = true
	sw.Banner = ""

	if err := sw.WriteNode(root); err != nil {
		rsp.Error = fmt.Sprintf("sw.WriteNode failed with %s", err)
		httpOkWithJSONCompact(w, r, rsp)
		return
	}
	d, err := format.Source(res.Bytes())
	if err != nil {
		fmt.Printf("format.Source() failed with %s\n", err)
		d = res.Bytes()
	}
	rsp.Go = string(d)
	httpOkWithJSONCompact(w, r, rsp)
}

// /xmltogo/*
// TODO: redirect to https://blog.kowalczyk.info/tools/xmltogo/
func handleXMLToGoIndex(w http.ResponseWriter, r *http.Request) {
	uri := r.URL.Path
	file := strings.TrimPrefix(uri, "/xmltogo/")
	if file == "" {
		file = "index.html"
	}
	path := filepath.Join("xmltogo", file)
	serveRelativeFile(w, r, path)
}
