package main

import (
	"io/fs"
	"net/http"
	"path"
	"path/filepath"
	"strings"
)

var (
	allArticles *Articles
	allTagURLS  []string // first item is tag, second is its url
	articleURLS []string // the order is the same as allArticles.articles
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

func serveStartHTML(w http.ResponseWriter, r *http.Request) {
	if r != nil {
		w.Header().Add("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK) // 200
	}
}
func serverGet(uri string) func(w http.ResponseWriter, r *http.Request) {
	logf(ctx(), "serverGet: '%s'\n", uri)
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
			serveStartHTML(w, r)
			genIndex(allArticles, w)
		}
	case "/archives.html":
		return func(w http.ResponseWriter, r *http.Request) {
			logf(ctx(), "serverGet: will serve '%s' with '%s'\n", uri, "writeArticlesArchiveForTag")
			serveStartHTML(w, r)
			writeArticlesArchiveForTag(allArticles, "", w)
		}
	case "/book/go-cookbook.html":
		return func(w http.ResponseWriter, r *http.Request) {
			logf(ctx(), "serverGet: will serve '%s' with '%s'\n", uri, "genGoCookbook")
			serveStartHTML(w, r)
			genGoCookbook(allArticles, w)
		}
	case "/changelog.html":
		return func(w http.ResponseWriter, r *http.Request) {
			logf(ctx(), "serverGet: will serve '%s' with '%s'\n", uri, "genChangelog")
			serveStartHTML(w, r)
			genChangelog(allArticles, w)
		}
	}

	n := len(articleURLS)
	uriLC := strings.ToLower(uri)
	for i := 0; i < n; i++ {
		if uriLC == articleURLS[i] {
			article := allArticles.articles[i]
			return func(w http.ResponseWriter, r *http.Request) {
				logf(ctx(), "serverGet: will serve '%s' with '%s'\n", uri, "genArticle")
				serveStartHTML(w, r)
				genArticle(article, w)
			}
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

func getURLSForFiles(startDir string, urlPrefix string) []string {
	var res []string
	filepath.WalkDir(startDir, func(filePath string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		dir := strings.TrimPrefix(filePath, startDir)
		uri := path.Join(urlPrefix, dir, d.Name())
		res = append(res, uri)
		return nil
	})
	return res
}

func serverURLS() []string {
	files := []string{"/index.html", "/archives.html", "/book/go-cookbook.html", "/changelog.html"}
	// TODO: filter out templates etc.
	files = append(files, getURLSForFiles("www", "/")...)
	files = append(files, getURLSForFiles(filepath.Join("notion_cache", "files"), "/img")...)
	files = append(files, articleURLS...)
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
	for _, article := range store.articles {
		articleURLS = append(articleURLS, article.URL())
	}

	/*
		regenMd()
		readRedirects(articles)

		genAtom(store, nil)
		genAtomAll(store, nil)
	*/

	waitSignal := StartServer(server)
	waitSignal()
}
