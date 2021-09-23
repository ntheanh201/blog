package main

import (
	"net/http"
	"path/filepath"
	"strings"
)

var (
	allArticles *Articles
	allTagURLS  []string // first item is tag, second is its url
)

func serveFile(w http.ResponseWriter, r *http.Request, path string) {
	if r == nil {
		d := readFileMust(path)
		_, err := w.Write(d)
		must(err)
		return
	}
	http.ServeFile(w, r, path)
}

func tryServeFile(uri string, dir string) func(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(uri, "/")
	path := filepath.Join(dir, name)
	send := func(w http.ResponseWriter, r *http.Request) {
		logf(ctx(), "tryServeFile: serving '%s' with '%s'\n", uri, path)
		serveFile(w, r, path)
	}
	if fileExists(path) {
		logf(ctx(), "tryServeFile: will serve '%s' with '%s'\n", uri, path)
		return send
	}
	return nil
}

func serveImage(uri string) func(w http.ResponseWriter, r *http.Request) {
	uri = strings.TrimPrefix(uri, "/img/")
	dir := filepath.Join("notion_cache", "files")
	return tryServeFile(uri, dir)
}

func serverGet(uri string) func(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(uri, "/img/") {
		return serveImage(uri)
	}
	if serve := tryServeFile(uri, "www"); serve != nil {
		return serve
	}
	switch uri {
	case "/index.html":
		return func(w http.ResponseWriter, r *http.Request) {
			logf(ctx(), "serverGet: will serve '%s' with '%s'\n", uri, "genIndex")
			if r != nil {
				w.Header().Add("Content-Type", "text/html")
				w.WriteHeader(http.StatusOK) // 200
			}
			genIndex(allArticles, w)
		}
	case "/archives.html":
		return func(w http.ResponseWriter, r *http.Request) {
			logf(ctx(), "serverGet: will serve '%s' with '%s'\n", uri, "writeArticlesArchiveForTag")
			if r != nil {
				w.Header().Add("Content-Type", "text/html")
				w.WriteHeader(http.StatusOK) // 200
			}
			writeArticlesArchiveForTag(allArticles, "", w)
		}
	}
	for i := 0; i < len(allTagURLS); i += 2 {
		tagURL := allTagURLS[i+1]
		if uri == tagURL {
			tag := allTagURLS[i]
			return func(w http.ResponseWriter, r *http.Request) {
				logf(ctx(), "serverGet: will serve '%s' with '%s'\n", uri, "writeArticlesArchiveForTag")
				if r != nil {
					w.Header().Add("Content-Type", "text/html")
					w.WriteHeader(http.StatusOK) // 200
				}
				writeArticlesArchiveForTag(allArticles, tag, w)
			}
		}
	}
	return nil
}

func serverURLS() []string {
	files := []string{"/index.html", "/archives.html"}
	// TODO: add all files from "www" directory and "notion_cache/files"
	return files
}

func doRun() {
	logf(ctx(), "doRun\n")

	serveAll := NewDynamicHandler(serverGet, serverURLS)

	server := &ServerConfig{
		Handlers:  []Handler{serveAll},
		CleanURLS: true,
	}

	loadTemplates()
	cc := getNotionCachingClient()
	allArticles = loadArticles(cc)
	logf(ctx(), "got %d articless\n", len(allArticles.articles))

	store := allArticles
	tags := map[string]struct{}{}
	for _, article := range store.getBlogNotHidden() {
		for _, tag := range article.Tags {
			tags[tag] = struct{}{}
		}
	}
	for tag := range tags {
		tagURL := "/tag/" + tag // TODO: URL-escape?
		allTagURLS = append(allTagURLS, tag, tagURL)
	}
	/*
		regenMd()
		readRedirects(articles)
		generateHTML(articles)
	*/

	waitSignal := StartServer(server)
	waitSignal()
}
