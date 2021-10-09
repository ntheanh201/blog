package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"encoding/xml"
	"html/template"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
)

func serveFileMust(w http.ResponseWriter, r *http.Request, path string) {
	if r == nil {
		d := readFileMust(path)
		_, err := w.Write(d)
		must(err)
		return
	}
	http.ServeFile(w, r, path)
}

func acceptsGzip(r *http.Request) bool {
	// TODO: would be safer to split by ", "
	return r != nil && strings.Contains(r.Header.Get("Accept-Encoding"), "gzip")
}

func serveMaybeGzippedFile(w http.ResponseWriter, r *http.Request, path string) {
	logf(ctx(), "path: '%s'\n", path)
	if !pathExists(path) {
		http.NotFound(w, r)
		return
	}
	contentType := mimeTypeFromFileName(path)
	usesGzip := acceptsGzip(r)
	if usesGzip {
		if pathExists(path + ".gz") {
			path = path + ".gz"
		} else {
			usesGzip = false
		}
	}
	if len(contentType) > 0 {
		w.Header().Set("Content-Type", contentType)
	}
	// https://www.maxcdn.com/blog/accept-encoding-its-vary-important/
	// prevent caching non-gzipped version
	w.Header().Add("Vary", "Accept-Encoding")
	if usesGzip {
		w.Header().Set("Content-Encoding", "gzip")
	}
	d, err := ioutil.ReadFile(path)
	if err != nil {
		logerrf(ctx(), "ioutil.ReadFile('%s') failed with '%s'\n", path, err)
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(d)))
	w.WriteHeader(200)
	w.Write(d)
}
func serveRelativeFile(w http.ResponseWriter, r *http.Request, relativePath string) {
	path := filepath.Join("www", relativePath)
	serveMaybeGzippedFile(w, r, path)
}

func serveTemplate(w http.ResponseWriter, r *http.Request, relativePath string, data interface{}) {
	path := filepath.Join("www", relativePath)
	if !pathExists(path) {
		path = filepath.Join("www", "tools", relativePath)
	}
	if !pathExists(path) {
		http.NotFound(w, r)
		return
	}
	files := []string{path}
	tmpl := template.Must(template.ParseFiles(files...))
	name := filepath.Base(relativePath)
	var buf bytes.Buffer
	err := tmpl.ExecuteTemplate(&buf, name, data)
	panicIfErr(err)
	httpOkWithHTML(w, r, buf.Bytes())
}

func httpOkBytesWithContentType(w http.ResponseWriter, r *http.Request, contentType string, content []byte) {
	w.Header().Set("Content-Type", contentType)
	// https://www.maxcdn.com/blog/accept-encoding-its-vary-important/
	// prevent caching non-gzipped version
	w.Header().Add("Vary", "Accept-Encoding")
	if acceptsGzip(r) {
		w.Header().Set("Content-Encoding", "gzip")
		// Maybe: if len(content) above certain size, write as we go (on the other
		// hand, if we keep uncompressed data in memory...)
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		gz.Write(content)
		gz.Close()
		content = buf.Bytes()
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(content)))
	w.Write(content)
}

func httpOkWithHTML(w http.ResponseWriter, r *http.Request, d []byte) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(d)
}

func httpOkWithJSON(w http.ResponseWriter, r *http.Request, v interface{}) {
	b, err := json.MarshalIndent(v, "", "\t")
	if err != nil {
		// should never happen
		logerrf(ctx(), "json.MarshalIndent() failed with %q\n", err)
	}
	httpOkBytesWithContentType(w, r, "application/json", b)
}

func httpOkWithJSONCompact(w http.ResponseWriter, r *http.Request, v interface{}) {
	b, err := json.Marshal(v)
	if err != nil {
		// should never happen
		logerrf(ctx(), "json.MarshalIndent() failed with %q\n", err)
	}
	httpOkBytesWithContentType(w, r, "application/json", b)
}

func httpOkWithXML(w http.ResponseWriter, r *http.Request, v interface{}) {
	b, err := xml.MarshalIndent(v, "", "\t")
	if err != nil {
		// should never happen
		logerrf(ctx(), "xml.MarshalIndent() failed with %q\n", err)
	}
	httpOkBytesWithContentType(w, r, "text/xml", b)
}
