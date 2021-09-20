package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/aymerick/raymond"
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/parser"
)

const csDir = "cheatsheets"

var (
	limitCheatsheets = false

	whitelist = []string{"python3", "go"}
	blacklist = []string{"101", "absinthe", "analytics.js", "analytics", "angularjs", "appcache", "cheatsheet-styles", "deku@1", "enzyme@2", "figlet", "firefox", "go", "index", "index@2016", "ledger-csv", "ledger-examples", "ledger-format", "ledger-periods",
		"ledger-query", "ledger", "package", "phoenix-ecto@1.2", "phoenix-ecto@1.3", "phoenix@1.2", "python", "react@0.14", "README", "vue@1.0.28"}
)

func init() {
	if !limitCheatsheets {
		whitelist = nil
	}
}

func newCsMarkdownParser() *parser.Parser {
	extensions := parser.NoIntraEmphasis |
		parser.Tables |
		parser.FencedCode |
		parser.Autolink |
		parser.Strikethrough |
		parser.SpaceHeadings |
		parser.AutoHeadingIDs |
		parser.HeadingIDs |
		parser.NoEmptyLineBeforeBlock
	return parser.NewWithExtensions(extensions)
}

func csBuildToc(doc ast.Node, path string) []*tocNode {
	//logf("csBuildToc: %s\n", path)
	//ast.Print(os.Stdout, doc)

	taken := map[string]bool{}
	ensureUniqueID := func(id string) {
		panicIf(taken[id], "duplicate heading id '%s' in '%s'", id, path)
		taken[id] = true
	}

	var currHeading *ast.Heading
	var currHeadingContent string
	var allHeaders []*tocNode
	var currToc *tocNode
	ast.WalkFunc(doc, func(node ast.Node, entering bool) ast.WalkStatus {
		switch v := node.(type) {
		case *ast.Heading:
			if entering {
				currHeading = v
			} else {
				ensureUniqueID(currHeading.HeadingID)
				tn := &tocNode{
					heading:      currHeading,
					Content:      currHeadingContent,
					ID:           currHeading.HeadingID,
					HeadingLevel: currHeading.Level,
				}
				allHeaders = append(allHeaders, tn)
				currToc = tn
				currHeading = nil
				currHeadingContent = ""
				//headingLevel := currHeading.Level
			}
		case *ast.Text:
			// the only child of ast.Heading is ast.Text (I think)
			if currHeading != nil && entering {
				currHeadingContent = string(v.Literal)
			} else {
				if entering && currToc != nil {
					currToc.SiblingsCount++
				}
			}
		default:
			if entering && currToc != nil {
				currToc.SiblingsCount++
			}
		}
		return ast.GoToNext
	})

	if false {
		for _, tn := range allHeaders {
			logf("h%d #%s %s %d siblings\n", tn.HeadingLevel, tn.heading.HeadingID, tn.Content, tn.SiblingsCount)
		}
	}
	cloneNode := func(n *tocNode) *tocNode {
		// clone but without children
		return &tocNode{
			heading:       n.heading,
			Content:       n.Content,
			HeadingLevel:  n.HeadingLevel,
			ID:            n.ID,
			SiblingsCount: n.SiblingsCount,
		}
	}

	buildToc := func() []*tocNode {
		first := cloneNode(allHeaders[0])
		toc := []*tocNode{first}
		stack := []*tocNode{first}
		for _, node := range allHeaders[1:] {
			node = cloneNode(node)
			stackLastIdx := len(stack) - 1
			curr := stack[stackLastIdx]
			currLevel := curr.HeadingLevel
			nodeLevel := node.HeadingLevel
			if nodeLevel > currLevel {
				// this is a child
				// TODO: should synthesize if we skip more than 1 level?
				panicIf(nodeLevel-currLevel > 1, "skipping more than 1 level in %s, '%s'", path, node.Content)
				curr.Children = append(curr.Children, node)
				stack = append(stack, node)
				curr = node
			} else if nodeLevel == currLevel {
				// this is a sibling, make current and attach to
				stack[stackLastIdx] = node
				if stackLastIdx > 0 {
					parent := stack[stackLastIdx-1]
					parent.Children = append(parent.Children, node)
				} else {
					toc = append(toc, node)
				}
			} else {
				// nodeLevel < currLevel
				for stackLastIdx > 0 {
					if stackLastIdx == 1 {
						toc = append(toc, node)
						stack = []*tocNode{node}
						stackLastIdx = 0
					} else {
						stack = stack[:stackLastIdx]
						stackLastIdx--
						curr = stack[stackLastIdx]
						if curr.HeadingLevel == nodeLevel {
							stack[stackLastIdx] = node
							parent := stack[stackLastIdx-1]
							parent.Children = append(parent.Children, node)
							stackLastIdx = 0
						}
					}
				}
			}
		}
		// remove intro if at the top level
		for i, node := range toc {
			if node.ID == "intro" && len(node.Children) == 0 {
				toc = append(toc[:i], toc[i+1:]...)
				return toc
			}
		}
		return toc
	}
	toc := buildToc()
	if false {
		printToc(toc, 0)
	}
	return toc
}

func printToc(nodes []*tocNode, indent int) {
	indentStr := func(indent int) string {
		return "............................"[:indent]
	}
	hdrStr := func(level int) string {
		return "#################"[:level]
	}

	for _, n := range nodes {
		s := indentStr(indent)
		hdr := hdrStr(n.HeadingLevel)
		logf("%s%s %s\n", s, hdr, n.Content)
		printToc(n.Children, indent+1)
	}
}

var reg *regexp.Regexp

func init() {
	reg = regexp.MustCompile(`{:.*}`)
}

func cleanupMarkdown(md []byte) []byte {
	s := string(md)
	// TODO: implement support of this in markdown parser
	// remove lines like: {: data-line="1"}
	s = reg.ReplaceAllString(s, "")
	s = strings.Replace(s, "{% raw %}", "", -1)
	s = strings.Replace(s, "{% endraw %}", "", -1)
	prev := s
	for prev != s {
		prev = s
		s = strings.Replace(s, "\n\n", "\n", -1)
	}
	return []byte(s)
}

type cheatSheet struct {
	fileNameBase string // unique name from file name, without extension
	mdFileName   string // path relative to www/cheatsheets directory
	mdPath       string
	htmlFullPath string
	// TODO: rename htmlFileName
	PathHTML   string // path relative to www/cheatsheets directory
	mdWithMeta []byte
	md         []byte
	meta       map[string]string
	html       []byte
	Title      string
}

func extractCheatSheetMetadata(cs *cheatSheet) {
	md := normalizeNewlines(cs.mdWithMeta)
	lines := strings.Split(string(md), "\n")
	// skip empty lines at the beginning
	for len(lines[0]) == 0 {
		lines = lines[1:]
	}
	if lines[0] != "---" {
		// no metadata
		cs.md = []byte(strings.Join(lines, "\n"))
		return
	}
	metaLines := []string{}
	lines = lines[1:]
	for lines[0] != "---" {
		metaLines = append(metaLines, lines[0])
		lines = lines[1:]
	}
	lines = lines[1:]
	cs.md = []byte(strings.Join(lines, "\n"))
	//logf("meta for '%s':\n%s\n", cs.mdPath, strings.Join(metaLines, "\n"))
	lastName := ""
	for _, line := range metaLines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 1 {
			s := strings.TrimSpace(parts[0])
			s = strings.Trim(s, `"`)
			v := cs.meta[lastName]
			if len(v) > 0 {
				v = v + "\n"
			}
			v += s
			cs.meta[lastName] = v
		} else {
			name := parts[0]
			s := strings.TrimSpace(parts[1])
			s = strings.Trim(s, `"`)
			s = strings.TrimLeft(s, "|")
			cs.meta[name] = s
			lastName = name
		}
	}
	cs.Title = cs.meta["title"]
	if cs.Title == "" {
		cs.Title = cs.fileNameBase
	}
}

type tocNode struct {
	heading *ast.Heading // not set if synthesized

	Content      string
	HeadingLevel int
	TocLevel     int
	ID           string

	SiblingsCount int

	Children []*tocNode // level of child is > our level

	tocHTML      []byte
	tocHTMLBlock *ast.HTMLBlock
	seen         bool
}

func genTocHTML(node *tocNode, level int) {
	nChildren := len(node.Children)
	buildToc := func() {
		shouldBuild := ((level >= 2) || (node.SiblingsCount == 0))
		if nChildren == 0 || !shouldBuild {
			return
		}

		s := `<div class="toc-mini">`
		for i, c := range node.Children {
			s += fmt.Sprintf(`<a href="#%s">%s</a>`, c.ID, c.Content)
			if i < nChildren-1 {
				s += `<span class="tmb">&bull;</span>`
			}
		}
		s += `</div>`
		//logf("genTocHTML: generating for %s '%s'\n", node.ID, s)
		node.tocHTML = []byte(s)
		node.tocHTMLBlock = &ast.HTMLBlock{Leaf: ast.Leaf{Literal: []byte(s)}}
	}
	buildToc()

	for _, c := range node.Children {
		genTocHTML(c, level+1)
	}
}

func findTocNodeForHeading(toc []*tocNode, h *ast.Heading) *tocNode {
	for _, n := range toc {
		if n.heading == h {
			return n
		}
		// deapth first search
		if c := findTocNodeForHeading(n.Children, h); c != nil {
			return c
		}
	}
	return nil
}

// for 2nd+ level headings we need to create a toc-mini pointing to its children
func insertAutoToc(doc ast.Node, toc []*tocNode) {

	for _, n := range toc {
		genTocHTML(n, 1)
	}

	// doc is ast.Document, all ast.Heading are direct childre
	// we fish out the ast.Heading and insert tocHTMLBlock after ast.Heading
	onceMore := true
	for onceMore {
		onceMore = false
		a := doc.GetChildren()
		for i, n := range a {
			hn, ok := n.(*ast.Heading)
			if !ok {
				continue
			}
			tn := findTocNodeForHeading(toc, hn)
			if tn == nil || tn.seen {
				continue
			}
			tn.seen = true
			if tn.tocHTMLBlock != nil {
				//logf("inserting toc for heading %s\n", hn.HeadingID)
				insertAstNodeChild(doc, tn.tocHTMLBlock, i+1)
				tn.tocHTMLBlock = nil
				// re-do from beginning if modified
				onceMore = true
			}
		}
	}
}

// insertAstNodeChild appends child to children of parent
// It panics if either node is nil.
func insertAstNodeChild(parent ast.Node, child ast.Node, i int) {
	//ast.RemoveFromTree(child)
	child.SetParent(parent)
	a := parent.GetChildren()
	if i >= len(a) {
		a = append(a, child)
	} else {
		a = append(a[:i], append([]ast.Node{child}, a[i:]...)...)
	}
	parent.SetChildren(a)
}

func buildFlatToc(toc []*tocNode, tocLevel int) []*tocNode {
	res := []*tocNode{}
	for _, n := range toc {
		n.TocLevel = tocLevel
		res = append(res, n)
		sub := buildFlatToc(n.Children, tocLevel+1)
		res = append(res, sub...)
	}
	return res
}

func processCheatSheet(cs *cheatSheet) {
	//logf("processCheatSheet: '%s'\n", cs.mdPath)
	cs.mdWithMeta = readFileMust(cs.mdPath)
	extractCheatSheetMetadata(cs)

	//logf("csGenHTML: for '%s'\n", cs.mdPath)
	md := cleanupMarkdown(cs.md)
	parser := newCsMarkdownParser()

	doc := markdown.Parse(md, parser)
	toc := csBuildToc(doc, cs.mdPath)
	tocFlat := buildFlatToc(toc, 0)

	// [[text, text.toLowerCase(), id, tocLevel], ...]
	searchIndex := [][]interface{}{}
	for _, toc := range tocFlat {
		s := toc.Content
		v := []interface{}{s, strings.ToLower(s), toc.ID, toc.TocLevel}
		searchIndex = append(searchIndex, v)
	}

	insertAutoToc(doc, toc)
	//ast.Print(os.Stdout, doc)
	renderer := newMarkdownHTMLRenderer("")
	mdHTML := string(markdown.Render(doc, renderer))

	tpl := string(readFileMust(filepath.Join(csDir, "cheatsheet.tmpl.html")))

	// on windows mdFileName is a windows-style path so change to unix/url style
	mdFileName := strings.Replace(cs.mdFileName, `\`, "/", -1)

	searchIndexJSON, err := json.Marshal(searchIndex)
	must(err)

	ctx := map[string]interface{}{
		"toc": toc,
		//"tocflat":    tocFlat,
		"title":             cs.Title,
		"mdFileName":        mdFileName,
		"content":           mdHTML,
		"searchIndexStatic": string(searchIndexJSON),
	}
	cs.html = []byte(raymond.MustRender(tpl, ctx))

	//logf("Processed %s, html size: %d\n", cs.mdPath, len(cs.html))
}

func genIndexHTML(cheatsheets []*cheatSheet) string {
	// sort by title
	sort.Slice(cheatsheets, func(i, j int) bool {
		t1 := strings.ToLower(cheatsheets[i].Title)
		t2 := strings.ToLower(cheatsheets[j].Title)
		return t1 < t2
	})

	byCat := map[string][]*cheatSheet{}
	for _, cs := range cheatsheets {
		cat := cs.meta["category"]
		if cat == "" {
			continue
		}
		byCat[cat] = append(byCat[cat], cs)
	}

	// build toc for categories
	categories := []string{}
	for cat := range byCat {
		categories = append(categories, cat)
	}
	sort.Strings(categories)
	cats := []map[string]interface{}{}
	for _, category := range categories {
		v := map[string]interface{}{}
		v["category"] = category
		catMetas := byCat[category]
		v["cheatsheets"] = catMetas
		cats = append(cats, v)
	}

	tpl := string(readFileMust(filepath.Join(csDir, "index.tmpl.html")))
	ctx := map[string]interface{}{
		"cheatsheets":      cheatsheets,
		"CheatsheetsCount": len(cheatsheets),
		"categories":       cats,
	}
	s := raymond.MustRender(tpl, ctx)
	return s
}

func genCheatSheetFiles() map[string][]byte {
	cheatsheets := []*cheatSheet{}

	isBlacklisted := func(s string, a []string) bool {
		s = strings.ToLower(s)
		for _, s2 := range a {
			if s == strings.ToLower(s2) {
				return true
			}
		}
		return false
	}

	isWhitelisted := func(s string, a []string) bool {
		if a == nil {
			return true
		}
		s = strings.ToLower(s)
		for _, s2 := range a {
			if s == strings.ToLower(s2) {
				return true
			}
		}
		return false
	}

	readFromDir := func(subDir string, blacklist []string, whitelist []string) {
		dir := filepath.Join(csDir, subDir)
		files, err := os.ReadDir(dir)
		must(err)
		for _, f := range files {
			if f.IsDir() {
				continue
			}
			name := f.Name()
			if filepath.Ext(name) != ".md" {
				continue
			}
			baseName := strings.Split(name, ".")[0]
			if isBlacklisted(baseName, blacklist) {
				//logf("blacklisted %s\n", f.Name())
				continue
			}
			if !isWhitelisted(baseName, whitelist) {
				//logf("!whitelisted %s\n", f.Name())
				continue
			}
			cs := &cheatSheet{
				fileNameBase: baseName,
				mdPath:       filepath.Join(dir, name),
				mdFileName:   filepath.Join(subDir, name),
				meta:         map[string]string{},
			}
			//logf("%s\n", cs.mdPath)
			cheatsheets = append(cheatsheets, cs)
		}
	}

	readFromDir("devhints", blacklist, whitelist)
	readFromDir("good", nil, whitelist)

	{
		// uniquify names
		taken := map[string]bool{}
		for _, cs := range cheatsheets {
			name := cs.fileNameBase
			n := 0
			for taken[name] {
				n++
				name = fmt.Sprintf("%s%d", cs.fileNameBase, n)
			}
			taken[name] = true
			cs.fileNameBase = name
		}
	}

	for _, cs := range cheatsheets {
		cs.PathHTML = cs.fileNameBase + ".html"
		cs.htmlFullPath = filepath.Join(csDir, cs.PathHTML)
	}

	logf("%d cheatsheets\n", len(cheatsheets))

	nThreads := runtime.NumCPU()
	//nThreads := 1
	sem := make(chan bool, nThreads)
	var wg sync.WaitGroup
	for _, cs := range cheatsheets {
		wg.Add(1)
		sem <- true
		go func(cs *cheatSheet) {
			processCheatSheet(cs)
			wg.Done()
			<-sem
		}(cs)
	}
	wg.Wait()
	files := map[string][]byte{}
	{
		path := filepath.Join(csDir, "cheatsheet.js")
		name := filepath.Join("s", "cheatsheet.js")
		files[name] = readFileMust(path)
	}
	{
		path := filepath.Join(csDir, "cheatsheet.css")
		name := filepath.Join("s", "cheatsheet.css")
		files[name] = readFileMust(path)
	}
	for _, cs := range cheatsheets {
		d := cs.html
		name := filepath.Base(cs.htmlFullPath)
		files[name] = d
	}
	files["index.html"] = []byte(genIndexHTML(cheatsheets))
	return files
}

func previewCheatSheets() {
	files := genCheatSheetFiles()
	uri := uploadFilesToInstantPreviewMust(files)
	openBrowser(uri)
}

func genCheatSheets(outDir string) {
	files := genCheatSheetFiles()
	for fileName, d := range files {
		path := filepath.Join("cheatsheets", fileName)
		wwwWriteFile(path, d)
	}
}
