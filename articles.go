package main

import (
	"html/template"
	"sort"
	"time"

	"github.com/kjk/notionapi"
)

var (
	notionBlogsStartPage      = "300db9dc27c84958a08b8d0c37f4cfe5"
	notionWebsiteStartPage    = "568ac4c064c34ef6a6ad0b8d77230681"
	notionGoCookbookStartPage = "7495260a1daa46118858ad2e049e77e6"
)

// Articles has info about all articles downloaded from notion
type Articles struct {
	idToArticle map[string]*Article
	idToPage    map[string]*notionapi.Page
	// all downloaded articles
	articles []*Article
	// articles that are not hidden
	articlesNotHidden []*Article
	// articles that belong to a blog
	blog []*Article
	// blog articles that are not hidden
	blogNotHidden []*Article
}

func (a *Articles) getNotHidden() []*Article {
	if a.articlesNotHidden == nil {
		var arr []*Article
		for _, article := range a.articles {
			if !article.IsHidden() {
				arr = append(arr, article)
			}
		}
		a.articlesNotHidden = arr
	}
	return a.articlesNotHidden
}

func (a *Articles) getBlogNotHidden() []*Article {
	if a.blogNotHidden == nil {
		var arr []*Article
		for _, article := range a.blog {
			if !article.IsHidden() {
				arr = append(arr, article)
			}
		}
		a.blogNotHidden = arr
	}
	return a.blogNotHidden
}

func buildArticleNavigation(article *Article, isRootPage func(string) bool, idToBlock map[string]*notionapi.Block) {
	// some already have path (e.g. those that belong to a collection)
	if len(article.Paths) > 0 {
		return
	}

	page := article.page.Root()
	currID := normalizeID(page.ParentID)

	var paths []URLPath
	for !isRootPage(currID) {
		block := idToBlock[currID]
		if block == nil {
			break
		}
		// parent could be a column
		if block.Type != notionapi.BlockPage {
			currID = normalizeID(block.ParentID)
			continue
		}
		title := block.Title
		uri := "/article/" + normalizeID(block.ID) + "/" + urlify(title)
		path := URLPath{
			Name: title,
			URL:  uri,
		}
		paths = append(paths, path)
		currID = normalizeID(block.ParentID)
	}

	// set in reverse order
	n := len(paths)
	for i := 1; i <= n; i++ {
		path := paths[n-i]
		article.Paths = append(article.Paths, path)
	}
}

var normalizeID = notionapi.ToNoDashID

func addIDToBlock(block *notionapi.Block, idToBlock map[string]*notionapi.Block) {
	id := normalizeID(block.ID)
	idToBlock[id] = block
	for _, block := range block.Content {
		if block == nil {
			continue
		}
		addIDToBlock(block, idToBlock)
	}
}

// build navigation bread-crumbs for articles
func buildArticlesNavigation(articles *Articles) {
	idToBlock := map[string]*notionapi.Block{}
	for _, a := range articles.articles {
		page := a.page
		if page == nil {
			continue
		}
		addIDToBlock(page.Root(), idToBlock)
	}

	isRoot := func(id string) bool {
		id = notionapi.ToNoDashID(id)
		switch id {
		case notionBlogsStartPage, notionWebsiteStartPage, notionGoCookbookStartPage:
			return true
		}
		return false
	}

	for _, article := range articles.articles {
		buildArticleNavigation(article, isRoot, idToBlock)
	}
}

func loadArticles(d *notionapi.CachingClient) *Articles {
	{
		timeStart := time.Now()
		d.PreLoadCache()
		logf(ctx(), "d.PreLoadCache() finished in %s\n", time.Since(timeStart))
	}

	res := &Articles{}
	nFromCache := 0
	nReq := 0
	timeStart := time.Now()
	afterDl := func(di *notionapi.DownloadInfo) error {
		if flgVerbose {
			return nil
		}
		nReq++
		dur := time.Since(timeStart)
		if !di.FromCache {
			logf(ctx(), "DL    %s %d, total time: %s\n", di.Page.ID, nReq, dur)
		} else {
			nFromCache++
			if nFromCache == 1 || nFromCache%16 == 0 {
				logf(ctx(), "CACHE %s %d, total time: %s\n", di.Page.ID, nReq, dur)
			}
		}
		return nil
	}
	_, err := d.DownloadPagesRecursively(notionWebsiteStartPage, afterDl)
	must(err)
	res.idToPage = map[string]*notionapi.Page{}
	for id, cp := range d.IdToCachedPage {
		page := cp.PageFromServer
		if page == nil {
			page = cp.PageFromCache
		}
		if page == nil {
			continue
		}
		res.idToPage[id] = page
	}

	res.idToArticle = map[string]*Article{}
	for id, page := range res.idToPage {
		panicIf(id != notionapi.ToNoDashID(id), "bad id '%s' sneaked in", id)
		article := notionPageToArticle(d, page)
		if article.urlOverride != "" {
			logvf("url override: %s => %s\n", article.urlOverride, article.ID)
		}
		res.idToArticle[id] = article
		// this might be legacy, short id. If not, we just set the same value twice
		articleID := article.ID
		res.idToArticle[articleID] = article
		if article.IsBlog() {
			res.blog = append(res.blog, article)
		}
		res.articles = append(res.articles, article)
	}

	for _, article := range res.articles {
		html, images := notionToHTML(d, article, res)
		article.BodyHTML = string(html)
		article.HTMLBody = template.HTML(article.BodyHTML)
		article.Images = append(article.Images, images...)
	}

	buildArticlesNavigation(res)

	sort.Slice(res.blog, func(i, j int) bool {
		return res.blog[i].PublishedOn.After(res.blog[j].PublishedOn)
	})

	return res
}

// MonthArticle combines article and a month
type MonthArticle struct {
	*Article
	DisplayMonth string
}

// Year describes articles in a given year
type Year struct {
	Name     string
	Articles []MonthArticle
}

// DisplayTitle returns a title for an article
func (a *MonthArticle) DisplayTitle() string {
	if a.Title != "" {
		return a.Title
	}
	return "no title"
}

// NewYear creates a new Year
func NewYear(name string) *Year {
	return &Year{Name: name, Articles: make([]MonthArticle, 0)}
}

func buildYearsFromArticles(articles []*Article) []Year {
	res := make([]Year, 0)
	var currYear *Year
	var currMonthName string
	n := len(articles)
	for i := 0; i < n; i++ {
		a := articles[i]
		yearName := a.PublishedOn.Format("2006")
		if currYear == nil || currYear.Name != yearName {
			if currYear != nil {
				res = append(res, *currYear)
			}
			currYear = NewYear(yearName)
			currMonthName = ""
		}
		ma := MonthArticle{Article: a}
		monthName := a.PublishedOn.Format("01")
		if monthName != currMonthName {
			ma.DisplayMonth = a.PublishedOn.Format("January 2")
		} else {
			ma.DisplayMonth = a.PublishedOn.Format("2")
		}
		currMonthName = monthName
		currYear.Articles = append(currYear.Articles, ma)
	}
	if currYear != nil {
		res = append(res, *currYear)
	}
	return res
}

func filterArticlesByTag(articles []*Article, tag string, include bool) []*Article {
	res := make([]*Article, 0)
	for _, a := range articles {
		hasTag := false
		for _, t := range a.Tags {
			if tag == t {
				hasTag = true
				break
			}
		}
		if include && hasTag {
			res = append(res, a)
		} else if !include && !hasTag {
			res = append(res, a)
		}
	}
	return res
}
