package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var (
	allArticles *Articles
	allTagURLS  []string // first item is tag, second is its url
	articleURLS []string // the order is the same as allArticles.articles
)

func tryServeFile(uri string, dir string) func(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(uri, "/")
	path := filepath.Join(dir, name)
	send := func(w http.ResponseWriter, r *http.Request) {
		logf(ctx(), "tryServeFile: serving '%s' with '%s'\n", uri, path)
		serveFile(w, r, path)
	}
	if fileExists(path) {
		//logf(ctx(), "tryServeFile: will serve '%s' with '%s'\n", uri, path)
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
	//logf(ctx(), "serverGet: '%s'\n", uri)
	store := allArticles
	if strings.HasPrefix(uri, "/img/") {
		return serveImage(uri)
	}
	if serve := tryServeFile(uri, "www"); serve != nil {
		return serve
	}
	writeData := func(w http.ResponseWriter, d []byte, err error) {
		must(err)
		_, err = w.Write(d)
		must(err)
	}
	switch uri {
	case "/index.html":
		return func(w http.ResponseWriter, r *http.Request) {
			//logf(ctx(), "serverGet: will serve '%s' with '%s'\n", uri, "genIndex")
			serveStart(w, r, uri)
			genIndex(store, w)
		}
	case "/archives.html":
		return func(w http.ResponseWriter, r *http.Request) {
			//logf(ctx(), "serverGet: will serve '%s' with '%s'\n", uri, "writeArticlesArchiveForTag")
			serveStart(w, r, uri)
			writeArticlesArchiveForTag(store, "", w)
		}
	case "/book/go-cookbook.html", "/articles/go-cookbook.html":
		return func(w http.ResponseWriter, r *http.Request) {
			//logf(ctx(), "serverGet: will serve '%s' with '%s'\n", uri, "genGoCookbook")
			serveStart(w, r, uri)
			genGoCookbook(store, w)
		}
	case "/changelog.html":
		return func(w http.ResponseWriter, r *http.Request) {
			//logf(ctx(), "serverGet: will serve '%s' with '%s'\n", uri, "genChangelog")
			serveStart(w, r, uri)
			genChangelog(store, w)
		}
	case "/sitemap.xml":
		return func(w http.ResponseWriter, r *http.Request) {
			//logf(ctx(), "serverGet: will serve '%s' with '%s'\n", uri, "genSiteMap")
			serveStart(w, r, uri)
			d, err := genSiteMap(store, "https://blog.kowalczyk.info")
			writeData(w, d, err)
		}
	case "/atom.xml":
		return func(w http.ResponseWriter, r *http.Request) {
			//logf(ctx(), "serverGet: will serve '%s' with '%s'\n", uri, "genAtomXML")
			serveStart(w, r, uri)
			d, err := genAtomXML(store, true)
			writeData(w, d, err)
		}
	case "/atom-all.xml":
		return func(w http.ResponseWriter, r *http.Request) {
			//logf(ctx(), "serverGet: will serve '%s' with '%s'\n", uri, "genAtomXML")
			serveStart(w, r, uri)
			d, err := genAtomXML(store, false)
			writeData(w, d, err)
		}
	case "/tools/generate-unique-id.html":
		return func(w http.ResponseWriter, r *http.Request) {
			//logf(ctx(), "serverGet: will serve '%s' with '%s'\n", uri, "genAtomXML")
			serveStart(w, r, uri)
			genToolGenerateUniqueID(store, w)
		}
	case "/404.html":
		return func(w http.ResponseWriter, r *http.Request) {
			//logf(ctx(), "serverGet: will serve '%s' with '%s'\n", uri, "gen404")
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
				//logf(ctx(), "serverGet: will serve '%s' with '%s'\n", uri, "genArticle")
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
				//logf(ctx(), "serverGet: will serve '%s' with '%s'\n", uri, "writeArticlesArchiveForTag")
				serveStart(w, r, uri)
				writeArticlesArchiveForTag(allArticles, tag, w)
			}
		}
	}
	return nil
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
		"/tools/generate-unique-id.html",
	}
	files = append(files, articleURLS...)
	return files
}

func makeDynamicServer() *ServerConfig {
	loadTemplates()

	serveAll := NewDynamicHandler(serverGet, serverURLS)

	// TODO: filter out templates etc.
	serveWWW := NewDirHandler("www", "/", nil)
	serveNotionImages := NewDirHandler(filepath.Join("notion_cache", "files"), "/img", nil)

	server := &ServerConfig{
		Handlers:  []Handler{serveWWW, serveNotionImages, serveAll},
		Port:      9001,
		CleanURLS: true,
	}

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
		tagURL := "/tag/" + tag + ".html" // TODO: URL-escape?
		allTagURLS = append(allTagURLS, tag, tagURL)
	}
	for _, article := range store.articles {
		uri := article.URL()
		articleURLS = append(articleURLS, uri)
	}
	return server
}

func genHTMLServer(dir string) {
	os.RemoveAll(generatedHTMLDir)
	regenMd()
	server := makeDynamicServer()
	WriteServerFilesToDir(generatedHTMLDir, server.Handlers)
}

func runServer() {
	logf(ctx(), "runServer\n")

	server := makeDynamicServer()
	waitSignal := StartServer(server)
	waitSignal()
}
