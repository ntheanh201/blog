package main

import (
	"crypto/sha1"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kjk/notionapi"
)

var (
	cacheDir = "notion_cache"
)

func sha1OfLink(link string) string {
	link = strings.ToLower(link)
	h := sha1.New()
	h.Write([]byte(link))
	return fmt.Sprintf("%x", h.Sum(nil))
}

var imgFiles []os.FileInfo

func findImageInDir(imgDir string, sha1 string) string {
	if len(imgFiles) == 0 {
		imgFiles, _ = ioutil.ReadDir(imgDir)
	}
	for _, fi := range imgFiles {
		if strings.HasPrefix(fi.Name(), sha1) {
			return filepath.Join(imgDir, fi.Name())
		}
	}
	return ""
}

func guessExt(fileName string, contentType string) string {
	ext := strings.ToLower(filepath.Ext(fileName))
	// TODO: maybe allow every non-empty extension. This
	// white-listing might not be a good idea
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp", ".tiff", ".svg":
		return ext
	}

	contentType = strings.ToLower(contentType)
	switch contentType {
	case "image/png":
		return ".png"
	case "image/jpeg":
		return ".jpg"
	case "image/svg+xml":
		return ".svg"
	}
	panic(fmt.Errorf("didn't find ext for file '%s', content type '%s'", fileName, contentType))
}

func downloadImage(c *notionapi.CachingClient, uri string, block *notionapi.Block) ([]byte, string, error) {
	resp, err := c.DownloadFile(uri, block)
	if err != nil {
		return nil, "", err
	}
	contentType := resp.Header.Get("Content-Type")
	// TODO: sniff from content
	ext := guessExt(uri, contentType)
	return resp.Data, ext, nil
}

// return path of cached image on disk
func downloadAndCacheImage(c *notionapi.CachingClient, uri string, block *notionapi.Block) (string, error) {
	sha := sha1OfLink(uri)

	//ext := strings.ToLower(filepath.Ext(uri))

	imgDir := filepath.Join(cacheDir, "img")
	err := os.MkdirAll(imgDir, 0755)
	must(err)

	cachedPath := findImageInDir(imgDir, sha)
	if cachedPath != "" {
		logvf("Image %s already downloaded as %s\n", uri, cachedPath)
		return cachedPath, nil
	}

	timeStart := time.Now()
	logf("Downloading %s ... ", uri)

	imgData, ext, err := downloadImage(c, uri, block)
	must(err)

	cachedPath = filepath.Join(imgDir, sha+ext)

	err = ioutil.WriteFile(cachedPath, imgData, 0644)
	if err != nil {
		return "", err
	}
	logf("finished in %s. Wrote as '%s'\n", time.Since(timeStart), cachedPath)

	return cachedPath, nil
}

/*
func rmFile(path string) {
	err := os.Remove(path)
	if err != nil {
		logf("os.Remove(%s) failed with %s\n", path, err)
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
	logf("Downloaded %s %s\n", pageID, page.Root().Title)
	return notionPageToArticle(d, page)
}
