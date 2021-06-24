package main

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/kjk/notionapi"
	"github.com/kjk/u"
)

var (
	destDir = "netlify_static"
)

func copyCSS() {
	src := filepath.Join("www", "css", "main.css")
	dst := filepath.Join(destDir, "main.css")
	u.CopyFileMust(dst, src)
}

func createDestDir() {
	err := os.MkdirAll(destDir, 0755)
	must(err)
}

func createNotionDirs() {
	err := os.MkdirAll(cacheDir, 0755)
	must(err)
}

// downloads and html
func testNotionToHTMLOnePage(d *notionapi.CachingClient, id string) {

	//id := "c9bef0f1c8fe40a2bc8b06ace2bd7d8f" // tools page, columns
	//id := "0a66e6c0c36f4de49417a47e2c40a87e" // mono-spaced page with toggle, devlog 2018
	//id := "484919a1647144c29234447ce408ff6b" // test toggle
	//id := "88aee8f43620471aa9dbcad28368174c" // test image and gist
	loadTemplates()
	createNotionDirs()
	createDestDir()

	id = normalizeID(id)
	article := loadPageAsArticle(d, id)

	canonicalURL := netlifyRequestGetFullHost() + article.URL()
	model := struct {
		Article          *Article
		CanonicalURL     string
		CoverImage       string
		PageTitle        string
		TagsDisplay      string
		HeaderImageURL   string
		NotionEditURL    string
		Description      string
		TwitterShareURL  string
		FacebookShareURL string
		LinkedInShareURL string
	}{
		Article:          article,
		CanonicalURL:     canonicalURL,
		CoverImage:       article.HeaderImageURL,
		PageTitle:        article.Title,
		Description:      article.Description,
		TwitterShareURL:  makeTwitterShareURL(article),
		FacebookShareURL: makeFacebookShareURL(article),
		LinkedInShareURL: makeLinkedinShareURL(article),
	}
	if article.page != nil {
		id := normalizeID(article.page.ID)
		model.NotionEditURL = "https://notion.so/" + id
	}

	var buf bytes.Buffer
	err := templates.ExecuteTemplate(&buf, "article.tmpl.html", model)
	must(err)
	data := buf.Bytes()
	data = bytes.Replace(data, []byte("/css/main.css"), []byte("/main.css"), -1)

	path := filepath.Join(destDir, "index.html")
	err = ioutil.WriteFile(path, data, 0644)
	must(err)
	copyCSS()

	err = os.Chdir(destDir)
	must(err)

	go func() {
		time.Sleep(time.Second * 1)
		u.OpenBrowser("http://localhost:2015")
	}()
	runCaddy()
}
