package main

import (
	"html/template"
	"sort"
	"time"

	"github.com/kjk/notionapi"
)

var (
	notionBlogsStartPage   = "cbbc16640fc24a7a9fb24660356a4409"
	notionWebsiteStartPage = "68f077a6dfb346358f219875e80ea72c"
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

func buildArticleNavigation(article *Article, isRootPage func(string) bool, idToBlock map[string]*notionapi.Block, idToArticle map[string]*Article) {
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
		id := normalizeID(block.ID)
		articleForBlock := idToArticle[id]
		uri := articleForBlock.URL()
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
		case notionBlogsStartPage, notionWebsiteStartPage:
			return true
		}
		return false
	}

	for _, article := range articles.articles {
		buildArticleNavigation(article, isRoot, idToBlock, articles.idToArticle)
	}
}

func loadArticles(d *notionapi.CachingClient) *Articles {
	{
		timeStart := time.Now()
		//TODO-ntheanh201: d.PreLoadCache()
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

	page := CollectionViewToPages(d)
	//example pages := []string{"cbbc16640fc24a7a9fb24660356a4409", "c484c3aea91a4e578cb783638b8fd6ac"}

	_, err := downloadPagesRecursively(d, page, afterDl)
	must(err)
	//res.idToPage = pages
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

func downloadPagesRecursively(c *notionapi.CachingClient, toVisit []string, afterDownload func(info *notionapi.DownloadInfo) error) (map[string]*notionapi.Page, error) {
	//toVisit := []*notionapi.NotionID{notionapi.NewNotionID(startPageID)}
	downloaded := map[string]*notionapi.Page{}
	for len(toVisit) > 0 {
		pageID := notionapi.ToDashID(toVisit[0])

		if pageID == notionapi.ToDashID(notionWebsiteStartPage) {
			toVisit = toVisit[1:]
			continue
		}

		toVisit = toVisit[1:]
		if downloaded[pageID] != nil {
			continue
		}
		nFromCache := c.RequestsFromCache
		nFromServer := c.RequestsFromServer
		timeStart := time.Now()
		page, err := c.DownloadPage(pageID)
		if err != nil {
			return nil, err
		}
		downloaded[pageID] = page
		if afterDownload != nil {
			di := &notionapi.DownloadInfo{
				Page:               page,
				RequestsFromCache:  c.RequestsFromCache - nFromCache,
				ReqeustsFromServer: c.RequestsFromServer - nFromServer,
				Duration:           time.Since(timeStart),
				FromCache:          c.RequestsFromServer == 0,
			}
			err = afterDownload(di)
			if err != nil {
				return nil, err
			}
		}

		//subPages := page.GetSubPages()
		subPages := getSubPages(page)
		toVisit = append(toVisit, subPages...)
	}
	n := len(downloaded)
	if n == 0 {
		return nil, nil
	}
	var ids []string
	for id := range downloaded {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	pages := make(map[string]*notionapi.Page, n)
	for i, id := range ids {
		pages[string(rune(i))] = downloaded[id]
	}
	return pages, nil
}

// GetSubPages return list of ids for direct sub-pages of this page
func getSubPages(p *notionapi.Page) []string {
	//if len(p.subPages) > 0 {
	//	return p.subPages
	//}
	root := p.Root()
	//panicIf(!isPageBlock(root))
	subPages := map[*notionapi.NotionID]struct{}{}
	seenBlocks := map[string]struct{}{}
	var blocksToVisit []*notionapi.NotionID
	for _, id := range root.ContentIDs {
		nid := notionapi.NewNotionID(id)
		blocksToVisit = append(blocksToVisit, nid)
	}
	for len(blocksToVisit) > 0 {
		nid := blocksToVisit[0]
		id := nid.DashID
		blocksToVisit = blocksToVisit[1:]
		if _, ok := seenBlocks[id]; ok {
			continue
		}
		seenBlocks[id] = struct{}{}
		block := p.BlockByID(nid)
		if p.IsSubPage(block) {
			subPages[nid] = struct{}{}
		}
		// need to recursively scan blocks with children
		for _, id := range block.ContentIDs {
			nid := notionapi.NewNotionID(id)
			blocksToVisit = append(blocksToVisit, nid)
		}
	}
	var res []string
	for id := range subPages {
		res = append(res, id.DashID)
	}
	sort.Strings(res)
	//sort.Slice(res, func(i, j int) bool {
	//	return res[i].DashID < res[j].DashID
	//})
	//p.subPages = res
	return res
}
