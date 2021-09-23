package main

import (
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/kjk/notionapi"
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

func serveStart(w http.ResponseWriter, r *http.Request, uri string) {
	if r == nil {
		return
	}
	ct := mimeTypeFromFileName(uri)
	w.Header().Add("Content-Type", ct)
	w.WriteHeader(http.StatusOK) // 200
}

func serverGet(uri string) func(w http.ResponseWriter, r *http.Request) {
	logf(ctx(), "serverGet: '%s'\n", uri)
	store := allArticles
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
			serveStart(w, r, uri)
			genIndex(store, w)
		}
	case "/archives.html":
		return func(w http.ResponseWriter, r *http.Request) {
			logf(ctx(), "serverGet: will serve '%s' with '%s'\n", uri, "writeArticlesArchiveForTag")
			serveStart(w, r, uri)
			writeArticlesArchiveForTag(store, "", w)
		}
	case "/book/go-cookbook.html", "/articles/go-cookbook.html":
		return func(w http.ResponseWriter, r *http.Request) {
			logf(ctx(), "serverGet: will serve '%s' with '%s'\n", uri, "genGoCookbook")
			serveStart(w, r, uri)
			genGoCookbook(store, w)
		}
	case "/changelog.html":
		return func(w http.ResponseWriter, r *http.Request) {
			logf(ctx(), "serverGet: will serve '%s' with '%s'\n", uri, "genChangelog")
			serveStart(w, r, uri)
			genChangelog(store, w)
		}
	case "/sitemap.xml":
		return func(w http.ResponseWriter, r *http.Request) {
			logf(ctx(), "serverGet: will serve '%s' with '%s'\n", uri, "genSiteMap")
			d, err := genSiteMap(store, "https://blog.kowalczyk.info")
			must(err)
			serveStart(w, r, uri)
			_, err = w.Write(d)
			must(err)
		}
	case "/atom.xml":
		return func(w http.ResponseWriter, r *http.Request) {
			logf(ctx(), "serverGet: will serve '%s' with '%s'\n", uri, "genAtomXML")
			d, err := genAtomXML(store, true)
			must(err)
			serveStart(w, r, uri)
			_, err = w.Write(d)
			must(err)
		}
	case "/atom-all.xml":
		return func(w http.ResponseWriter, r *http.Request) {
			logf(ctx(), "serverGet: will serve '%s' with '%s'\n", uri, "genAtomXML")
			d, err := genAtomXML(store, false)
			must(err)
			serveStart(w, r, uri)
			_, err = w.Write(d)
			must(err)
		}
	case "/404.html":
		return func(w http.ResponseWriter, r *http.Request) {
			logf(ctx(), "serverGet: will serve '%s' with '%s'\n", uri, "gen404")
			serveStart(w, r, uri)
			gen404(store, w)
		}
	}

	n := len(articleURLS)
	//uriLC := strings.ToLower(uri)
	for i := 0; i < n; i++ {
		if uri == articleURLS[i] {
			article := allArticles.articles[i]
			return func(w http.ResponseWriter, r *http.Request) {
				logf(ctx(), "serverGet: will serve '%s' with '%s'\n", uri, "genArticle")
				serveStart(w, r, uri)
				genArticle(article, w)
			}
		}
	}

	n = len(allTagURLS)
	for i := 0; i < n; i += 2 {
		tagURL := allTagURLS[i+1]
		if uri == tagURL {
			tag := allTagURLS[i]
			return func(w http.ResponseWriter, r *http.Request) {
				logf(ctx(), "serverGet: will serve '%s' with '%s'\n", uri, "writeArticlesArchiveForTag")
				serveStart(w, r, uri)
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
		dir = filepath.ToSlash(dir)
		dir = strings.TrimPrefix(dir, "/")
		uri := path.Join(urlPrefix, dir)
		//logf("getURLSForFiles: dir: '%s'\n", dir)
		res = append(res, uri)
		return nil
	})
	return res
}

func serverURLS() []string {
	files := []string{
		"/index.html",
		"/archives.html",
		"/book/go-cookbook.html",
		"/articles/go-cookbook.html",
		"/changelog.html",
		"/sitemap.xml",
		"/atom.xml",
		"/atom-all.xml",
		"/404.html",
		"/software/index.html",
	}
	// TODO: filter out templates etc.
	files = append(files, getURLSForFiles("www", "/")...)
	files = append(files, getURLSForFiles(filepath.Join("notion_cache", "files"), "/img")...)
	files = append(files, articleURLS...)
	return files
}

func makeDynamicServer() *ServerConfig {
	loadTemplates()

	serveAll := NewDynamicHandler(serverGet, serverURLS)

	server := &ServerConfig{
		Handlers:  []Handler{serveAll},
		CleanURLS: true,
	}

	cachingPolicy = notionapi.PolicyCacheOnly
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
		uri := article.URL() // TODO: change in the metadata
		if uri == "/software/" {
			uri = "/software/index.html"
		}
		articleURLS = append(articleURLS, uri)
	}
	return server
}

func genHTMLServer(dir string) {
	os.RemoveAll(generatedHTMLDir)
	server := makeDynamicServer()
	WriteServerFilesToDir(generatedHTMLDir, server.Handlers)
}

func runServer() {
	logf(ctx(), "runServer\n")

	server := makeDynamicServer()
	waitSignal := StartServer(server)
	waitSignal()
}
