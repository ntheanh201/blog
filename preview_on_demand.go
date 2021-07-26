package main

import (
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/kjk/u"
)

var (
	gPreviewArticles *Articles
)

func serve404(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusNotFound)
	gen404(nil, w)
}

// tempRedirect gives a Moved Permanently response.
func doRedirect(w http.ResponseWriter, r *http.Request, newPath string, code int) {
	if q := r.URL.RawQuery; q != "" {
		newPath += "?" + q
	}
	w.Header().Set("Location", newPath)
	if code == 0 {
		code = http.StatusFound
	}
	w.WriteHeader(code)
}

func handleIndexOnDemand(w http.ResponseWriter, r *http.Request) {
	logf("uri: %s\n", r.URL.Path)
	uri := r.URL.Path
	nAgain := 0
again:
	nAgain++
	if nAgain > 3 {
		logf("Too many 200 redirects for '%s'\n", r.URL.Path)
		serve404(w, r)
		return
	}
	if tryServeKnown(w, r) {
		return
	}

	if tryServeNotionCacheImg(w, r) {
		return
	}

	if tryServeTagArchive(w, r) {
		return
	}

	if tryServeArticle(w, r) {
		return
	}

	for _, redir := range wwwRedirects {
		if redir.from == uri {
			// this is internal rewrite
			if redir.code == 200 {
				logf("Rewrite %s => %s\n", uri, redir.to)
				uri = redir.to
				r.URL.Path = uri
				goto again
			}
			logf("Redirect %s => %s, code: %d\n", uri, redir.to, redir.code)
			doRedirect(w, r, redir.to, redir.code)
		}
	}

	isDir := strings.HasSuffix(uri, "/")
	fileName := filepath.FromSlash(uri[1:])
	if isDir {
		fileName = filepath.Join(fileName, "index.html")
	}
	path := filepath.Join("www", fileName)
	if !u.FileExists(path) {
		logf("path '%s' for url '%s' doesn't exist\n", path, uri)
		serve404(w, r)
		return
	}
	logf("Serving file: %s\n", path)
	http.ServeFile(w, r, path)
}

var knownURLs = map[string]func(*Articles, io.Writer) error{
	"/":                              genIndex,
	"/changelog.html":                genChangelog,
	"/archives.html":                 genArchives,
	"/sitemap.xml":                   genSitemap,
	"/atom.xml":                      genAtom,
	"/404.html":                      gen404,
	"/atom-all.xml":                  genAtomAll,
	"/book/go-cookbook.html":         genGoCookbook,
	"/tools/generate-unique-id.html": genToolGenerateUniqueID,
	"/tools/generate-unique-id":      genToolGenerateUniqueID,
}

func tryServeKnown(w http.ResponseWriter, r *http.Request) bool {
	uri := r.URL.Path
	if fn := knownURLs[uri]; fn != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusOK)
		logIfError(fn(gPreviewArticles, w))
		return true
	}
	return false
}

func findArticleByID(store *Articles, id string) *Article {
	for _, article := range store.articles {
		if article.ID == id {
			return article
		}
	}
	return nil
}

func tryServeArticle(w http.ResponseWriter, r *http.Request) bool {
	uri := r.URL.Path
	articlePath := strings.TrimPrefix(uri, "/article/")
	if uri == articlePath {
		return false
	}
	articlePath = strings.Split(articlePath, "/")[0]
	articleID := strings.TrimSuffix(articlePath, ".html")
	article := findArticleByID(gPreviewArticles, articleID)
	if article == nil {
		logf("Didn't find article with id '%s' for uri '%s'\n", articleID, uri)
		return false
	}
	genArticle(article, w)
	return true
}

// tag/<tagname>
func tryServeTagArchive(w http.ResponseWriter, r *http.Request) bool {
	uri := r.URL.Path
	tag := strings.TrimPrefix(uri, "/tag/")
	if uri == tag {
		return false
	}
	writeArticlesArchiveForTag(gPreviewArticles, tag, w)
	return true
}

// for /img/<name> check notion_cache/img
func tryServeNotionCacheImg(w http.ResponseWriter, r *http.Request) bool {
	uri := r.URL.Path
	imgName := strings.TrimPrefix(uri, "/img/")
	if uri == imgName {
		return false
	}
	path := filepath.Join("notion_cache", "img", imgName)
	if !u.FileExists(path) {
		return false
	}
	http.ServeFile(w, r, path)
	return true
}

// https://blog.gopheracademy.com/advent-2016/exposing-go-on-the-internet/
func makeHTTPServerOnDemand() *http.Server {
	mux := &http.ServeMux{}

	mux.HandleFunc("/", handleIndexOnDemand)

	srv := &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  120 * time.Second, // introduced in Go 1.8
		Handler:      mux,
	}
	return srv
}

func startPreviewOnDemand(articles *Articles) {
	gPreviewArticles = articles

	addAllRedirects(gPreviewArticles)

	httpSrv := makeHTTPServerOnDemand()
	httpSrv.Addr = "127.0.0.1:8183"

	go func() {
		err := httpSrv.ListenAndServe()
		// mute error caused by Shutdown()
		if err == http.ErrServerClosed {
			err = nil
		}
		must(err)
		logf("HTTP server shutdown gracefully\n")
	}()
	logf("Started listening on %s\n", httpSrv.Addr)
	u.OpenBrowser("http://" + httpSrv.Addr)

	u.WaitForCtrlC()
}
