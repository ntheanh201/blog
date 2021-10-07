package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var mimeTypes = map[string]string{
	// not present in mime.TypeByExtension()
	".txt": "text/plain",
	".exe": "application/octet-stream",
}

func mimeTypeFromFileName(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	ct := mimeTypes[ext]
	if ct == "" {
		ct = mime.TypeByExtension(ext)
	}
	if ct == "" {
		// if all else fails
		ct = "application/octet-stream"
	}
	return ct
}

// can be used for http.Get() requests with better timeouts. New one must be created
// for each Get() request
func newTimeoutClient(connectTimeout time.Duration, readWriteTimeout time.Duration) *http.Client {
	timeoutDialer := func(cTimeout time.Duration, rwTimeout time.Duration) func(net, addr string) (c net.Conn, err error) {
		return func(netw, addr string) (net.Conn, error) {
			conn, err := net.DialTimeout(netw, addr, cTimeout)
			if err != nil {
				return nil, err
			}
			conn.SetDeadline(time.Now().Add(rwTimeout))
			return conn, nil
		}
	}

	return &http.Client{
		Transport: &http.Transport{
			Dial:  timeoutDialer(connectTimeout, readWriteTimeout),
			Proxy: http.ProxyFromEnvironment,
		},
	}
}

func httpGet(url string) ([]byte, error) {
	// default timeout for http.Get() is really long, so dial it down
	// for both connection and read/write timeouts
	timeoutClient := newTimeoutClient(time.Second*120, time.Second*120)
	resp, err := timeoutClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("'%s': status code not 200 (%d)", url, resp.StatusCode)
	}
	return ioutil.ReadAll(resp.Body)
}

func httpPost(uri string, body []byte) ([]byte, error) {
	// default timeout for http.Get() is really long, so dial it down
	// for both connection and read/write timeouts
	timeoutClient := newTimeoutClient(time.Second*120, time.Second*120)
	resp, err := timeoutClient.Post(uri, "", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("'%s': status code not 200 (%d)", uri, resp.StatusCode)
	}
	return ioutil.ReadAll(resp.Body)
}

func createMultiPartForm(form map[string]string) (string, io.Reader, error) {
	body := new(bytes.Buffer)
	mp := multipart.NewWriter(body)
	defer mp.Close()
	for key, val := range form {
		if strings.HasPrefix(val, "@") {
			val = val[1:]
			file, err := os.Open(val)
			if err != nil {
				return "", nil, err
			}
			defer file.Close()
			part, err := mp.CreateFormFile(key, val)
			if err != nil {
				return "", nil, err
			}
			io.Copy(part, file)
		} else {
			mp.WriteField(key, val)
		}
	}
	return mp.FormDataContentType(), body, nil
}

func httpPostMultiPart(uri string, files map[string]string) ([]byte, error) {
	contentType, body, err := createMultiPartForm(files)
	if err != nil {
		return nil, err
	}
	// default timeout for http.Get() is really long, so dial it down
	// for both connection and read/write timeouts
	timeoutClient := newTimeoutClient(time.Second*120, time.Second*120)
	resp, err := timeoutClient.Post(uri, contentType, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("'%s': status code not 200 (%d)", uri, resp.StatusCode)
	}
	return ioutil.ReadAll(resp.Body)
}

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
		logerrf(nil, "ioutil.ReadFile('%s') failed with '%s'\n", path, err)
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

func httpOkWithText(w http.ResponseWriter, s string) {
	w.Header().Set("Content-Type", "text/plain")
	io.WriteString(w, s)
}

func httpOkWithHTML(w http.ResponseWriter, r *http.Request, d []byte) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(d)
}

func httpOkWithJSON(w http.ResponseWriter, r *http.Request, v interface{}) {
	b, err := json.MarshalIndent(v, "", "\t")
	if err != nil {
		// should never happen
		logerrf(nil, "json.MarshalIndent() failed with %q\n", err)
	}
	httpOkBytesWithContentType(w, r, "application/json", b)
}

func httpOkWithXML(w http.ResponseWriter, r *http.Request, v interface{}) {
	b, err := xml.MarshalIndent(v, "", "\t")
	if err != nil {
		// should never happen
		logerrf(nil, "xml.MarshalIndent() failed with %q\n", err)
	}
	httpOkBytesWithContentType(w, r, "text/xml", b)
}
