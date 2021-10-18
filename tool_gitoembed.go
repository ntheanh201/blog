package main

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"
)

/*
Notes about oembed standard:
- https://oembed.com/
- https://blog.ycombinator.com/how-to-build-an-oembed-integration-for-your-startup-and-why-its-necessary/
*/

var (
	oembedDownloadCache = NewHTTPDownloadCache()
	//baseURL             = "https://www.onlinetool.io"
	baseURL = "https://blog.kowalczyk.info"
)

// json oembed response
// example response: https://api.clyp.it/oembed?url=https%3a%2f%2fclyp.it%2fzhwuptos&format=JSON
type oembedResult struct {
	Version      int64       `json:"version"`
	Type         string      `json:"type"`
	ProviderName string      `json:"provider_name"`
	ProviderURL  string      `json:"provider_url"`
	Height       interface{} `json:"height"` // can be a number like 256 or string like "100%"
	Width        interface{} `json:"width"`
	Title        string      `json:"title"`
	HTML         string      `json:"html"`
}

/*
<link rel="alternate" type="text/xml+oembed"
  href="http://flickr.com/services/oembed?url=http%3A%2F%2Fflickr.com%2Fphotos%2Fbees%2F2362225867%2F&format=xml"
  title="Bacon Lollys oEmbed Profile" />
*/
type oembedXMLResult struct {
	XMLName      xml.Name    `xml:"oembed"`
	Version      string      `xml:"version"`
	Type         string      `xml:"type"`
	Width        interface{} `xml:"width"`
	Height       interface{} `xml:"height"` // can be a number like 256 or string like "100%"
	Title        string      `xml:"title"`
	URL          string      `xml:"url,omitempty"`
	ProviderName string      `json:"provider_name"`
	ProviderURL  string      `json:"provider_url"`
	HTML         string      `json:"html"`
}

// covnert
// https://github.com/essentialbooks/books/blob/master/netlify.toml
// =>
// https://raw.githubusercontent.com/essentialbooks/books/master/netlify.toml
// returns empty string if not a valid format
func getGitHubRawURL(uri string) string {
	uri = strings.TrimPrefix(uri, "http://")
	uri = strings.TrimPrefix(uri, "https://")
	// this is because when copy&paste Chrome removes one of the "//"
	uri = strings.TrimPrefix(uri, "https:/")
	uri = strings.TrimPrefix(uri, "http:/")
	parts := strings.SplitN(uri, "/", 5)
	if len(parts) < 5 {
		return ""
	}
	if parts[0] != "github.com" {
		return ""
	}
	if parts[3] != "blob" {
		return ""
	}
	newParts := []string{"raw.githubusercontent.com", parts[1], parts[2], parts[4]}
	return "https://" + strings.Join(newParts, "/")
}

func getFileNameFromURL(uri string) string {
	parts := strings.Split(uri, "/")
	n := len(parts)
	if n == 1 {
		return uri
	}
	return parts[n-1]
}

type oembedQuery struct {
	url     string
	theme   string
	format  string
	noLines bool
}

func parseOembedQuery(uri *url.URL) *oembedQuery {
	res := &oembedQuery{}
	query := uri.Query()
	res.url = strings.TrimSpace(query.Get("url"))
	res.format = strings.TrimSpace(query.Get("format"))
	if _, ok := query["nolines"]; ok {
		res.noLines = true
	}
	res.theme = strings.TrimSpace(query.Get("theme"))
	return res
}

// /gitoembed/
// TODO: redirect to https://blog.kowalczyk.info/tools/gitoembed
func handleGitOembedIndex(w http.ResponseWriter, r *http.Request) {
	uri := r.URL.Path
	if uri != "/gitoembed/" {
		http.NotFound(w, r)
		return
	}
	relativePath := filepath.Join("gitoembed", "index.html")
	serveRelativeFile(w, r, relativePath)
}

// /gitoembed/widget?url=${GITURL}&nolines&theme=${theme}
// to test:
// http://127.0.0.1:8508/gitoembed/widget?https://github.com/kjk/programming-books.io/blob/master/books/go/0200-panic-and-recover/recover_from_panic.go?theme=github
func handleGitOembedWidget(w http.ResponseWriter, r *http.Request) {

	args := parseOembedQuery(r.URL)
	githubURL := args.url
	if githubURL == "" {
		http.NotFound(w, r)
		return
	}

	rawURL := getGitHubRawURL(githubURL)
	if rawURL == "" {
		// not a valid url
		http.NotFound(w, r)
		return
	}

	query := "&url=" + githubURL
	if args.noLines {
		query += "&nolines"
	}
	if args.theme != "" {
		query += "&theme" + args.theme
	}
	fileName := getFileNameFromURL(githubURL)

	timeStart := time.Now()
	d, fromCache, err := oembedDownloadCache.Download(rawURL)
	dur := time.Since(timeStart)
	ctx := context.Background()
	if err != nil {
		logf(ctx, "Failed to download %s. Time: %s. Error: %s\n", rawURL, dur, err)
		http.NotFound(w, r)
		return
	}
	if fromCache {
		logf(ctx, "Got %s (%d bytes) from cache in %s\n", rawURL, len(d), dur)
	} else {
		logf(ctx, "Downloaded %s (%d bytes) in %s\n", rawURL, len(d), dur)
	}
	var buf bytes.Buffer
	f := makeHTMLFormatter(args.noLines)
	theme := validateTheme(args.theme)
	err = codeHighlight(&buf, string(d), fileName, f, theme)
	if err != nil {
		logerrf(ctx, "quick.Highlight failed with %s\n", err)
		return
	}
	data := struct {
		Title          string
		OembedLinkJSON string
		OembedLinkXML  string
		Body           template.HTML
		FileName       string
		OriginalURL    string
		ServiceURL     string
	}{
		Title:          fileName,
		OembedLinkJSON: baseURL + "/gitoembed/oembed?format=json" + query,
		OembedLinkXML:  baseURL + "/gitoembed/oembed?format=xml" + query,
		Body:           template.HTML(buf.Bytes()),
		FileName:       fileName,
		OriginalURL:    githubURL,
		ServiceURL:     baseURL + "/tools/gitoembed/",
	}
	path := filepath.Join("gitoembed", "oembed.tmpl.html")
	serveTemplate(w, r, path, data)
}

// /gitoembed/oembed?url=${url}&format=[json|xml]&nolines&theme=${theme}
func handleGitOembedOembed(w http.ResponseWriter, r *http.Request) {

	args := parseOembedQuery(r.URL)
	// url is mandatory
	gitHubURL := args.url
	if gitHubURL == "" {
		http.NotFound(w, r)
		return
	}
	format := "json"
	if strings.EqualFold(args.format, "xml") {
		format = "xml"
	}
	widgetURL := baseURL + "/gitoembed/widget?url=" + gitHubURL
	if args.noLines {
		widgetURL += "&nolines"
	}
	if args.theme != "" {
		widgetURL += "&theme=" + args.theme
	}

	// oembed spec requires those are integers. Those are somewhat arbitrary values
	width := 720
	height := 320
	html := fmt.Sprintf(`<iframe width="100%%" height=%v src="%s" frameborder="0" onload="resizeFrame(this);"></iframe>`, height, widgetURL)
	if format == "json" {
		resjs := oembedResult{
			Version:      1,
			Type:         "rich",
			ProviderName: "gitoembed",
			ProviderURL:  baseURL + "/gitoembed/",
			Height:       height,
			Width:        width,
			Title:        getFileNameFromURL(gitHubURL),
			HTML:         html,
		}
		httpOkWithJSON(w, r, resjs)
		return
	}

	resxml := oembedXMLResult{
		Version:      "1.0",
		Type:         "rich",
		ProviderName: "gitoembed",
		ProviderURL:  baseURL + "/gitoembed/",
		Height:       height,
		Width:        width,
		Title:        getFileNameFromURL(gitHubURL),
		HTML:         html,
	}
	httpOkWithXML(w, r, resxml)
}
