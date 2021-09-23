package main

import "net/http"

func serverGet(uri string) func(w http.ResponseWriter, r *http.Request) {
	return nil
}

func serverURLS() []string {
	return nil
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
	articles := loadArticles(cc)
	logf(ctx(), "got %d articless\n", len(articles.articles))

	/*
		regenMd()
		readRedirects(articles)
		generateHTML(articles)
	*/

	waitSignal := StartServer(server)
	waitSignal()
}
