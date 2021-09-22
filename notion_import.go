package main

import (
	"os"

	"github.com/kjk/notionapi"
)

var (
	cacheDir = "notion_cache"
)

var imgFiles []os.FileInfo

/*
func rmFile(path string) {
	err := os.Remove(path)
	if err != nil {
		logf(ctx(), "os.Remove(%s) failed with %s\n", path, err)
	}
}

func rmCached(pageID string) {
	id := normalizeID(pageID)
	rmFile(filepath.Join(cacheDir, id+".txt"))
}
*/

func loadPageAsArticle(d *notionapi.CachingClient, pageID string) *Article {
	page, err := d.DownloadPage(pageID)
	must(err)
	logf(ctx(), "Downloaded %s %s\n", pageID, page.Root().Title)
	return notionPageToArticle(d, page)
}
