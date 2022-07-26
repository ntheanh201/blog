package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

var articleRedirectsTxt = `
`

var redirects = [][]string{
	{"/index.html", "/"},
	{"/blog", "/"},
	{"/blog/", "/"},
	{"/feed/rss2/atom.xml", "/atom.xml"},
	{"/feed/rss2/", "/atom.xml"},
	{"/feed/rss2", "/atom.xml"},
	{"/feed/", "/atom.xml"},
	{"/feed", "/atom.xml"},
	{"/feedburner.xml", "/atom.xml"},
}

var articleRedirects = make(map[string]string)

func readRedirects(store *Articles) {
	d := []byte(articleRedirectsTxt)
	lines := bytes.Split(d, []byte{'\n'})
	for _, l := range lines {
		if len(l) == 0 {
			continue
		}
		parts := strings.Split(string(l), "|")
		panicIf(len(parts) != 2, "malformed article_redirects.txt, len(parts) = %d (!2)", len(parts))
		idStr := parts[0]
		url := strings.TrimSpace(parts[1])
		idNum, err := strconv.Atoi(idStr)
		panicIf(err != nil, "malformed line in article_redirects.txt. Line:\n%s\nError: %s\n", l, err)
		id := encodeBase64(idNum)
		a := store.idToArticle[id]
		if a != nil {
			articleRedirects[url] = id
			continue
		}
		//logvf("skipping redirect '%s' because article with id %d no longer present\n", string(l), id)
	}
}

var (
	wwwRedirects []*wwwRedirect
)

type wwwRedirect struct {
	from string
	to   string
	// valid code is 301, 302, 200, 404
	code int
}

func addRedirect(from, to string, code int) {
	r := wwwRedirect{
		from: from,
		to:   to,
		code: code,
	}
	wwwRedirects = append(wwwRedirects, &r)
}

func addRewrite(from, to string) {
	addRedirect(from, to, 200)
}

func addTempRedirect(from, to string) {
	addRedirect(from, to, 302)
}

func addStaticRedirects() {
	for _, redirect := range redirects {
		from := redirect[0]
		to := redirect[1]
		addTempRedirect(from, to)
	}
}

func addArticleRedirects(store *Articles) {
	readRedirects(store)
	for from, articleID := range articleRedirects {
		from = "/" + from
		article := store.idToArticle[articleID]
		panicIf(article == nil, "didn't find article for id '%s'", articleID)
		to := article.URL()
		addTempRedirect(from, to) // TODO: change to permanent
	}

}

// redirect /articles/:id/* => /articles/:id/pretty-title
const wwwRedirectsProlog = `/articles/:id/*	/articles/:id.html	200
`

func writeRedirects() {
	buf := bytes.NewBufferString(wwwRedirectsProlog)
	for _, r := range wwwRedirects {
		s := fmt.Sprintf("%s\t%s\t%d\n", r.from, r.to, r.code)
		buf.WriteString(s)
	}
	wwwWriteFile("_redirects", buf.Bytes())
}

type redirectInfo struct {
	URL  string
	Code int
}

func readRedirectsJSON() map[string]redirectInfo {
	res := map[string]redirectInfo{}
	d := readFileMust("redirects.json")
	var js map[string]interface{}
	must(json.Unmarshal(d, &js))
	for k, v := range js {
		a := v.([]interface{})
		uri := a[0].(string)
		code := (int)(a[1].(float64))
		ri := redirectInfo{
			URL:  uri,
			Code: code,
		}
		res[k] = ri
	}
	return res
}
