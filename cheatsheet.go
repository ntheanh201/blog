package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/parser"
)

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

type tocNode struct {
	heading *ast.Heading
	content string
	level   int

	children  []*tocNode // level of child is > our level
	nSiblings int
	parent    *tocNode
}

func csBuildToc(md []byte) string {
	logf("csBuildToc: printing heading, len(md): %d\n", len(md))
	parser := newCsMarkdownParser()
	doc := parser.Parse(md)
	//ast.Print(os.Stdout, doc)

	taken := map[string]bool{}
	makeUniqueID := func(id string) string {
		curr := id
		n := 0
		for taken[curr] {
			n++
			curr = fmt.Sprintf("%s%d", id, n)
		}
		taken[curr] = true
		return curr
	}

	var currHeading *ast.Heading
	var currHeadingContent string
	var toc []*tocNode
	var currToc *tocNode
	ast.WalkFunc(doc, func(node ast.Node, entering bool) ast.WalkStatus {
		switch v := node.(type) {
		case *ast.Heading:
			if entering {
				currHeading = v
			} else {
				currHeading.HeadingID = makeUniqueID(currHeading.HeadingID)
				tn := &tocNode{
					heading: currHeading,
					content: currHeadingContent,
					level:   currHeading.Level,
				}
				toc = append(toc, tn)
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
					currToc.nSiblings++
				}
			}
		default:
			if entering && currToc != nil {
				currToc.nSiblings++
			}
		}
		return ast.GoToNext
	})
	for _, tn := range toc {
		logf("h%d #%s %s %d siblings\n", tn.level, tn.heading.HeadingID, tn.content, tn.nSiblings)
	}

	return ""
}

func cleanupMarkdown(md []byte) []byte {
	s := string(md)
	// remove lines like: {: data-line="1"}
	//const reg = /{:.*}/g;
	//s = s.replace(reg, "");
	s = strings.Replace(s, "{% raw %}", "", -1)
	s = strings.Replace(s, "{% endraw %}", "", -1)
	prev := s
	for prev != s {
		prev = s
		s = strings.Replace(s, "\n\n", "\n", -1)
	}
	return []byte(s)
}

func csGenHTML(cs *cheatSheet) {
	logf("csGenHTML: for '%s'\n", cs.mdPath)
	md := cleanupMarkdown(cs.md)
	parser := newCsMarkdownParser()
	renderer := newMarkdownHTMLRenderer("")
	tocHTML := csBuildToc(md)
	content := string(markdown.ToHTML(md, parser, renderer))
	startHTML := `
  <div id="start"></div>
  <div id="wrapped-content"></div>
`
	innerHTML := tocHTML + startHTML + `<div id="content">` + "\n" + content + "\n" + `</div>`

	res := `<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8" />
	<title>${title} cheatsheet</title>
	<link href="s/main.css" rel="stylesheet" />
	<script src="s/main.js"></script>
</head>

<body onload="start()">
	<div class="breadcrumbs"><a href="/">Home</a> / <a href="index.html">cheatsheets</a> / ${title} cheatsheet</div>
	<div class="edit">
			<a href="https://github.com/kjk/blog/blob/master/www/cheatsheets/${mdFileName}" >edit</a>
	</div>
	${innerHTML}
</body>  
</html>`
	res = strings.Replace(res, "${innerHTML}", string(innerHTML), -1)
	// on windows mdFileName is a windows-style path so change to unix/url style
	mdFileName := strings.Replace(cs.mdFileName, `\`, "/", -1)
	res = strings.Replace(res, "${mdFileName}", mdFileName, -1)
	title := cs.meta["title"]
	if title == "" {
		title = cs.fileNameBase
	}
	res = strings.Replace(res, "${title}", title, -1)
	cs.html = []byte(res)
	logf("Processed %s, html size: %d\n", cs.mdPath, len(cs.html))
}

type cheatSheet struct {
	fileNameBase string // unique name from file name, without extension
	mdFileName   string // path relative to www/cheatsheet directory
	mdPath       string
	htmlFullPath string
	// TODO: rename htmlFileName
	pathHTML   string // path relative to www/cheatsheet directory
	mdWithMeta []byte
	md         []byte
	meta       map[string]string
	html       []byte
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
	logf("meta for '%s':\n%s\n", cs.mdPath, strings.Join(metaLines, "\n"))
	for _, line := range metaLines {
		parts := strings.SplitN(line, ":", 2)
		name := parts[0]
		cs.meta[name] = strings.TrimSpace(parts[1])
	}
}

func processCheatSheet(cs *cheatSheet) {
	cs.mdWithMeta = readFileMust(cs.mdPath)
	extractCheatSheetMetadata(cs)
	csGenHTML(cs)
}

func genIndexHTML(cheatsheets []*cheatSheet) string {
	for _, cs := range cheatsheets {
		title := cs.meta["title"]
		if title == "" {
			cs.meta["title"] = cs.fileNameBase
		}
	}

	// sort by title
	sort.Slice(cheatsheets, func(i, j int) bool {
		cs1 := cheatsheets[i]
		cs2 := cheatsheets[j]
		t1 := strings.ToLower(cs1.meta["title"])
		t2 := strings.ToLower(cs2.meta["title"])
		return t1 < t2
	})

	byCat := map[string][]*cheatSheet{}
	for _, cs := range cheatsheets {
		cat := cs.meta["category"]
		if cat == "" {
			continue
		}
		a := byCat[cat]
		a = append(a, cs)
		byCat[cat] = a
	}

	tocHTML := ""
	for _, cs := range cheatsheets {
		title := cs.meta["title"]
		s := `
<div class="index-toc-item with-bull"><a href="${pathHTML}">${title}</a></div>`
		s = strings.Replace(s, "${title}", title, -1)
		s = strings.Replace(s, "${pathHTML}", cs.pathHTML, -1)
		tocHTML += s
	}

	categories := []string{}
	for cat := range byCat {
		categories = append(categories, cat)
	}
	sort.Strings(categories)
	catsHTML := ""
	for _, category := range categories {
		catMetas := byCat[category]
		catHTML := `<div class="index-toc">`
		catHTML += strings.Replace(`<div> <b>${category}</b>:&nbsp;</div>`, "${category}", category, -1)
		for _, meta := range catMetas {
			s := `
<div class="with-bull"><a href="${pathHTML}">${title}</a></div>`
			s = strings.Replace(s, "${pathHTML}", meta.pathHTML, -1)
			s = strings.Replace(s, "${title}", meta.meta["title"], -1)
			catHTML += s
			catHTML += `</div>`
			catsHTML += catHTML
		}
	}
	nCheatsheets := strconv.Itoa(len(cheatsheets))
	s := `<!DOCTYPE html>
<html>

<head>
	<meta charset="utf-8" />
	<title>cheatsheets</title>
	<link href="s/main.css" rel="stylesheet" />
	<script src="//unpkg.com/alpinejs" defer></script>
	<script src="s/main.js"></script>
</head>

<body onload="startIndex()">
	<div class="breadcrumbs"><a href="/">Home</a> / cheatsheets</div>

	<div x-init="$watch('search', val => { filterList(val);})" x-data="{ search: '' }" class="input-wrapper">
		<div>${nCheatsheets} cheatsheets: <input placeholder="'/' to search" @keyup.escape="search=''" id="search-input" type="text" x-model="search"></div>
	</div>

	<div class="index-toc">
		${tocHTML}
	</div>
	<div class="by-topic"><center>By topic:</center></div>
	${catsHTML}
</body>
</html>
`
	s = strings.Replace(s, "${tocHTML}", tocHTML, -1)
	s = strings.Replace(s, "${nCheatsheets}", nCheatsheets, -1)
	s = strings.Replace(s, "${catsHTML}", catsHTML, -1)
	return s
}

func cheatsheets() {
	cheatsheets := []*cheatSheet{}

	readFromDir := func(subDir string) {
		dir := filepath.Join("www", "cheatsheets", subDir)
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
			if baseName != "go" {
				//logf("%s ", baseName)
				continue
			}
			cs := &cheatSheet{
				fileNameBase: baseName,
				mdPath:       filepath.Join(dir, name),
				mdFileName:   filepath.Join(subDir, name),
				meta:         map[string]string{},
			}
			logf("%s\n", cs.mdPath)
			cheatsheets = append(cheatsheets, cs)
		}
	}

	readFromDir("devhints")
	readFromDir("other")

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
			cs.fileNameBase = name
		}
	}

	for _, cs := range cheatsheets {
		cs.pathHTML = cs.fileNameBase + ".html"
		cs.htmlFullPath = filepath.Join("www", "cheatsheets", cs.pathHTML)
	}

	logf("%d cheatsheets\n", len(cheatsheets))

	sem := make(chan bool, runtime.NumCPU())
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

	// upload to instantpreview.dev
	files := map[string][]byte{}
	sDir := filepath.Join("www", "cheatsheets", "s")
	{
		path := filepath.Join(sDir, "main.js")
		name := filepath.Join("s", "main.js")
		files[name] = readFileMust(path)
	}
	{
		path := filepath.Join(sDir, "main.css")
		name := filepath.Join("s", "main.css")
		files[name] = readFileMust(path)
	}
	{
		path := filepath.Join("www", "cheatsheets", "index.html")
		name := "index.html"
		files[name] = readFileMust(path)
	}
	for _, cs := range cheatsheets {
		d := cs.html
		name := filepath.Base(cs.htmlFullPath)
		files[name] = d
	}
	files["index.html"] = []byte(genIndexHTML(cheatsheets))
	uri := uploadFilesToInstantPreviewMust(files)
	logf("uploaded %d cheatsheets under: %s\n", len(cheatsheets), uri)
	openBrowser(uri)
}
