package main

import (
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kjk/notionapi"
)

var (
	useCache = true
	destDir  = "notion_www"
)

// convert 2131b10c-ebf6-4938-a127-7089ff02dbe4 to 2131b10cebf64938a1277089ff02dbe4
func normalizeID(s string) string {
	return strings.Replace(s, "-", "", -1)
}

func openLogFileForPageID(pageID string) (io.WriteCloser, error) {
	name := fmt.Sprintf("%s.go.log.txt", pageID)
	path := filepath.Join(notionLogDir, name)
	f, err := os.Create(path)
	if err != nil {
		fmt.Printf("os.Create('%s') failed with %s\n", path, err)
		return nil, err
	}
	notionapi.Logger = f
	return f, nil
}

func articleFromPage(pageInfo *notionapi.PageInfo) *Article {
	blocks := pageInfo.Page.Content
	//fmt.Printf("extractMetadata: %s-%s, %d blocks\n", title, id, len(blocks))
	// metadata blocks are always at the beginning. They are TypeText blocks and
	// have only one plain string as content
	page := pageInfo.Page
	title := page.Title
	id := normalizeID(page.ID)
	article := &Article{
		pageInfo: pageInfo,
		Title:    title,
	}
	nBlock := 0
	var publishedOn time.Time
	var err error
	endLoop := false
	for len(blocks) > 0 {
		block := blocks[0]
		//fmt.Printf("  %d %s '%s'\n", nBlock, block.Type, block.Title)

		if block.Type != notionapi.BlockText {
			//fmt.Printf("extractMetadata: ending look because block %d is of type %s\n", nBlock, block.Type)
			break
		}

		if len(block.InlineContent) == 0 {
			//fmt.Printf("block %d of type %s and has no InlineContent\n", nBlock, block.Type)
			blocks = blocks[1:]
			break
		} else {
			//fmt.Printf("block %d has %d InlineContent\n", nBlock, len(block.InlineContent))
		}

		inline := block.InlineContent[0]
		// must be plain text
		if !inline.IsPlain() {
			//fmt.Printf("block: %d of type %s: inline has attributes\n", nBlock, block.Type)
			break
		}

		// remove empty lines at the top
		s := strings.TrimSpace(inline.Text)
		if s == "" {
			//fmt.Printf("block: %d of type %s: inline.Text is empty\n", nBlock, block.Type)
			blocks = blocks[2:]
			break
		}
		//fmt.Printf("  %d %s '%s'\n", nBlock, block.Type, s)

		parts := strings.SplitN(s, ":", 2)
		if len(parts) != 2 {
			//fmt.Printf("block: %d of type %s: inline.Text is not key/value. s='%s'\n", nBlock, block.Type, s)
			break
		}
		key := strings.ToLower(strings.TrimSpace(parts[0]))
		val := strings.TrimSpace(parts[1])
		switch key {
		case "tags":
			article.Tags = parseTags(val)
			//fmt.Printf("Tags: %v\n", res.Tags)
		case "id":
			articleSetID(article, val)
			//fmt.Printf("ID: %s\n", res.ID)

		case "publishedon":
			publishedOn, err = parseDate(val)
			panicIfErr(err)
		case "date", "createdat":
			article.PublishedOn, err = parseDate(val)
			panicIfErr(err)
		case "updatedat":
			article.UpdatedOn, err = parseDate(val)
			panicIfErr(err)
		case "status":
			setStatusMust(article, val)
		case "description":
			article.Description = val
			//fmt.Printf("Description: %s\n", res.Description)
		case "headerimage":
			setHeaderImageMust(article, val)
		case "collection":
			setCollectionMust(article, val)
		default:
			// assume that unrecognized meta means this article doesn't have
			// proper meta tags. It might miss meta-tags that are badly named
			endLoop = true
			/*
				rmCached(pageInfo.ID)
				title := pageInfo.Page.Title
				panicMsg("Unsupported meta '%s' in notion page with id '%s', '%s'", key, normalizeID(pageInfo.ID), title)
			*/
		}
		if endLoop {
			break
		}
		blocks = blocks[1:]
		nBlock++
	}
	pageInfo.Page.Content = blocks

	// PublishedOn over-writes Date and CreatedAt
	if !publishedOn.IsZero() {
		article.PublishedOn = publishedOn
	}

	if article.UpdatedOn.IsZero() {
		article.UpdatedOn = article.PublishedOn
	}

	if article.ID == "" {
		article.ID = id
	}

	article.Body = notionToHTML(pageInfo)
	article.BodyHTML = string(article.Body)
	article.HTMLBody = template.HTML(article.BodyHTML)

	return article
}

func notionToHTML(pageInfo *notionapi.PageInfo) []byte {
	gen := NewHTMLGenerator(pageInfo)
	return gen.Gen()
}

// TODO: change this to download from Notion via cache, so that
// notionRedownload() is just calling this, for consistency
func loadArticlesFromNotion() []*Article {
	pagesToIgnore := []string{
		notionBlogsStartPage, notionGoCookbookStartPage,
	}
	fileInfos, err := ioutil.ReadDir(cacheDir)
	panicIfErr(err)

	var res []*Article
	for _, fi := range fileInfos {
		if fi.IsDir() {
			continue
		}
		name := fi.Name()
		ext := filepath.Ext(name)
		if ext != ".json" {
			continue
		}
		ignorePage := false
		for _, s := range pagesToIgnore {
			if strings.Contains(name, s) {
				ignorePage = true
			}
		}
		if ignorePage {
			continue
		}
		parts := strings.Split(name, ".")
		pageID := parts[0]
		pageInfo := loadPageFromCache(pageID)
		article := articleFromPage(pageInfo)
		res = append(res, article)
	}

	return res
}

func rmFile(path string) {
	err := os.Remove(path)
	if err != nil {
		fmt.Printf("os.Remove(%s) failed with %s\n", path, err)
	}
}

func rmCached(pageID string) {
	id := normalizeID(pageID)
	rmFile(filepath.Join(notionLogDir, id+".go.log.txt"))
	rmFile(filepath.Join(cacheDir, id+".json"))
}

func copyCSS() {
	src := filepath.Join("www", "css", "main.css")
	dst := filepath.Join(destDir, "main.css")
	err := copyFile(dst, src)
	panicIfErr(err)
}

func createNotionDirs() {
	os.MkdirAll(notionLogDir, 0755)
	os.MkdirAll(cacheDir, 0755)
	os.MkdirAll(destDir, 0755)
	copyCSS()
}

// downloads and html
func testOneNotionPage() {
	//id := "c9bef0f1c8fe40a2bc8b06ace2bd7d8f" // tools page, columns
	//id := "0a66e6c0c36f4de49417a47e2c40a87e" // mono-spaced page with toggle, devlog 2018
	id := "484919a1647144c29234447ce408ff6b" // test toggle
	createNotionDirs()
	id = normalizeID(id)
	article, err := loadPageAsArticle(id)
	panicIfErr(err)
	path := filepath.Join(destDir, "index.html")
	d := notionToHTML(article.pageInfo)
	err = ioutil.WriteFile(path, d, 0644)
	panicIfErr(err)
}

func testNotionToHTML() {
	createNotionDirs()
	//notionapi.DebugLog = true
	startPageID := normalizeID(notionWebsiteStartPage)
	articles := loadNotionPages(startPageID)
	fmt.Printf("Loaded %d articles\n", len(articles))

	for _, article := range articles {
		id := normalizeID(article.ID)
		name := id + ".html"
		if id == startPageID {
			name = "index.html"
		}
		path := filepath.Join(destDir, name)
		d := notionToHTML(article.pageInfo)
		err := ioutil.WriteFile(path, d, 0644)
		panicIfErr(err)
	}
}