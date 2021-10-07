package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/kjk/cheatsheets/pkg/server"
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
		serveFileMust(w, r, path)
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
	n := len(allTagURLS)
	for i := 0; i < n; i += 2 {
		tagURL := allTagURLS[i+1]
		files = append(files, tagURL)
	}
	return files
}

func makeDynamicServer() *server.Server {
	loadTemplates()

	serveAll := server.NewDynamicHandler(serverGet, serverURLS)

	// TODO: filter out templates etc.
	serveWWW := server.NewDirHandler("www", "/", nil)
	serveNotionImages := server.NewDirHandler(filepath.Join("notion_cache", "files"), "/img", nil)

	server := &server.Server{
		Handlers:  []server.Handler{serveWWW, serveNotionImages, serveAll},
		Port:      httpPort,
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
	os.RemoveAll(dirWwwGenerated)
	regenMd()
	srv := makeDynamicServer()
	nFiles := 0
	totalSize := int64(0)
	onWritten := func(path string, d []byte) {
		fsize := int64(len(d))
		totalSize += fsize
		sizeStr := formatSize(fsize)
		if nFiles%256 == 0 {
			logf(ctx(), "generateStatic: file %d '%s' of size %s\n", nFiles+1, path, sizeStr)
		}
		nFiles++
	}
	server.WriteServerFilesToDir(dirWwwGenerated, srv.Handlers, onWritten)
}

func runServer() {
	logf(ctx(), "runServer\n")

	server := makeDynamicServer()
	waitSignal := StartServer(server)
	waitSignal()
}

func runServerProd() {
	panicIf(!dirExists(dirWwwGenerated))
	h := server.NewDirHandler(dirWwwGenerated, "/", nil)
	logf(ctx(), "runServerProd starting, hasSpacesCreds: %v, %d urls\n", hasSpacesCreds(), len(h.URLS()))
	srv := &server.Server{
		Handlers:  []server.Handler{h},
		CleanURLS: true,
		Port:      httpPort,
	}
	closeHTTPLog := openHTTPLog()
	defer closeHTTPLog() // TODO: this actually doesn't take in prod
	httpSrv := MakeHTTPServer(srv)
	logf(ctx(), "Starting server on http://%s'\n", httpSrv.Addr)
	if isWindows() {
		openBrowser(fmt.Sprintf("http://%s", httpSrv.Addr))
	}
	err := httpSrv.ListenAndServe()
	logf(ctx(), "runServerProd: httpSrv.ListenAndServe() returned '%s'\n", err)
}

func MakeHTTPServer(srv *server.Server) *http.Server {
	panicIf(srv == nil, "must provide srv")
	httpPort := 8080
	if srv.Port != 0 {
		httpPort = srv.Port
	}
	httpAddr := fmt.Sprintf(":%d", httpPort)
	if isWindows() {
		httpAddr = "localhost" + httpAddr
	}

	mainHandler := func(w http.ResponseWriter, r *http.Request) {
		//logf(ctx(), "mainHandler: '%s'\n", r.RequestURI)
		timeStart := time.Now()
		defer func() {
			if p := recover(); p != nil {
				logf(ctx(), "mainHandler: panicked with with %v\n", p)
				http.Error(w, fmt.Sprintf("Error: %v", r), http.StatusInternalServerError)
				logHTTPReq(r, http.StatusInternalServerError, 0, time.Since(timeStart))
				panic(p)
			}
		}()
		uri := r.URL.Path
		serve, _ := srv.FindHandler(uri)
		if serve == nil {
			http.NotFound(w, r)
			logHTTPReq(r, http.StatusNotFound, 0, time.Since(timeStart))
			return
		}
		if serve != nil {
			cw := server.CapturingResponseWriter{ResponseWriter: w}
			serve(&cw, r)
			logHTTPReq(r, cw.StatusCode, cw.Size, time.Since(timeStart))
			return
		}
		http.NotFound(w, r)
		logHTTPReq(r, http.StatusNotFound, 0, time.Since(timeStart))
	}

	httpSrv := &http.Server{
		ReadTimeout:  120 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  120 * time.Second, // introduced in Go 1.8
		Handler:      http.HandlerFunc(mainHandler),
	}
	httpSrv.Addr = httpAddr
	return httpSrv
}

// returns function that will wait for SIGTERM signal (e.g. Ctrl-C) and
// shutdown the server
func StartHTTPServer(httpSrv *http.Server) func() {
	logf(ctx(), "Starting server on http://%s'\n", httpSrv.Addr)
	if isWindows() {
		openBrowser(fmt.Sprintf("http://%s", httpSrv.Addr))
	}

	chServerClosed := make(chan bool, 1)
	go func() {
		err := httpSrv.ListenAndServe()
		// mute error caused by Shutdown()
		if err == http.ErrServerClosed {
			err = nil
		}
		must(err)
		logf(ctx(), "trying to shutdown HTTP server\n")
		chServerClosed <- true
	}()

	return func() {
		c := make(chan os.Signal, 2)
		signal.Notify(c, os.Interrupt /* SIGINT */, syscall.SIGTERM)

		sig := <-c
		logf(ctx(), "Got signal %s\n", sig)

		if httpSrv != nil {
			go func() {
				// Shutdown() needs a non-nil context
				_ = httpSrv.Shutdown(ctx())
			}()
			select {
			case <-chServerClosed:
				// do nothing
				logf(ctx(), "server shutdown cleanly\n")
			case <-time.After(time.Second * 5):
				// timeout
				logf(ctx(), "server killed due to shutdown timeout\n")
			}
		}
	}
}

func StartServer(srv *server.Server) func() {
	httpSrv := MakeHTTPServer(srv)
	return StartHTTPServer(httpSrv)
}
