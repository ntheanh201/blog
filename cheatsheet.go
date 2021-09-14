package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/parser"
)

const csDir = "cheatsheets"

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
	id      string

	children  []*tocNode // level of child is > our level
	nSiblings int
	//parent    *tocNode
}

func genTocHTML(toc []*tocNode) string {
	html := `<div class="toc">`
	for _, e := range toc {
		if len(e.children) == 0 {
			s := "\n" + `<a href="#${id}">${name}</a><br>`
			s = strings.Replace(s, "${id}", e.id, -1)
			s = strings.Replace(s, "${name}", e.content, -1)
			html += s
			continue
		}
		s := "\n" + `<b>${name}</b>: `
		s = strings.Replace(s, "${name}", e.content, -1)
		for i, te := range e.children {
			if i > 0 {
				s += ", "
			}
			tmp := `<a href="#${te[1]}">${te[0]}</a>`
			tmp = strings.Replace(tmp, "${te[1]}", te.id, -1)
			tmp = strings.Replace(tmp, "${te[0]}", te.content, -1)
			s += tmp
		}
		s += "<br>"
		html += s
	}
	html += "</div>\n\n"
	return html
}

func csBuildToc(md []byte) string {
	//logf("csBuildToc: printing heading, len(md): %d\n", len(md))
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
					id:      currHeading.HeadingID,
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

	if false {
		for _, tn := range toc {
			logf("h%d #%s %s %d siblings\n", tn.level, tn.heading.HeadingID, tn.content, tn.nSiblings)
		}
	}

	allHeaders := toc
	toc = nil
	var curr *tocNode
	for _, node := range allHeaders {
		if !(node.level == 2 || node.level == 3) {
			continue
		}
		if node.level == 2 {
			curr = node
			toc = append(toc, curr)
			continue
		}
		// must be h3
		if curr == nil {
			curr = &tocNode{
				content: "Main",
				id:      "main",
			}
			toc = append(toc, curr)
		}
		curr.children = append(curr.children, node)
	}
	return genTocHTML(toc)
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

func csGenHTML(cs *cheatSheet) {
	//logf("csGenHTML: for '%s'\n", cs.mdPath)
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

	res := string(readFileMust(filepath.Join(csDir, "cheatsheet.tmpl.html")))
	res = strings.Replace(res, "{{.InnerHTML}}", string(innerHTML), -1)
	// on windows mdFileName is a windows-style path so change to unix/url style
	mdFileName := strings.Replace(cs.mdFileName, `\`, "/", -1)
	res = strings.Replace(res, "{{.MdFileName}}", mdFileName, -1)
	res = strings.Replace(res, "{{.Title}}", cs.title, -1)
	cs.html = []byte(res)
	//logf("Processed %s, html size: %d\n", cs.mdPath, len(cs.html))
}

type cheatSheet struct {
	fileNameBase string // unique name from file name, without extension
	mdFileName   string // path relative to www/cheatsheets directory
	mdPath       string
	htmlFullPath string
	// TODO: rename htmlFileName
	pathHTML   string // path relative to www/cheatsheets directory
	mdWithMeta []byte
	md         []byte
	meta       map[string]string
	html       []byte
	title      string
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
	cs.title = cs.meta["title"]
	if cs.title == "" {
		cs.title = cs.fileNameBase
	}
}

func processCheatSheet(cs *cheatSheet) {
	//logf("processCheatSheet: '%s'\n", cs.mdPath)
	cs.mdWithMeta = readFileMust(cs.mdPath)
	extractCheatSheetMetadata(cs)
	csGenHTML(cs)
}

func genIndexHTML(cheatsheets []*cheatSheet) string {
	// sort by title
	sort.Slice(cheatsheets, func(i, j int) bool {
		t1 := strings.ToLower(cheatsheets[i].title)
		t2 := strings.ToLower(cheatsheets[j].title)
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

	tocHTML := ""
	for _, cs := range cheatsheets {
		s := `
<div class="index-toc-item with-bull"><a href="${pathHTML}">${title}</a></div>`
		s = strings.Replace(s, "${title}", cs.title, -1)
		s = strings.Replace(s, "${pathHTML}", cs.pathHTML, -1)
		tocHTML += s
	}

	// build toc for categories
	catsHTML := ""
	categories := []string{}
	for cat := range byCat {
		categories = append(categories, cat)
	}
	sort.Strings(categories)
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
		}
		catHTML += `</div>`
		catsHTML += catHTML
	}

	nCheatsheets := strconv.Itoa(len(cheatsheets))
	s := string(readFileMust(filepath.Join(csDir, "index.tmpl.html")))
	s = strings.Replace(s, "{{.TocHTML}}", tocHTML, -1)
	s = strings.Replace(s, "{{.CheatsheetsCount}}", nCheatsheets, -1)
	s = strings.Replace(s, "{{.CatsHTML}}", catsHTML, -1)
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

	readFromDir := func(subDir string, blacklist []string) {
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
				logf("blacklisted %s\n", f.Name())
				continue
			}
			if false && baseName != "go" {
				//logf("%s ", baseName)
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

	blacklist := []string{"101", "absinthe", "analytics.js", "analytics", "angularjs", "appcache", "cheatsheet-styles", "deku@1", "enzyme@2", "figlet", "firefox", "go", "index", "index@2016", "ledger-csv", "ledger-examples", "ledger-format", "ledger-periods",
		"ledger-query", "ledger", "package", "phoenix-ecto@1.2", "phoenix-ecto@1.3", "phoenix@1.2", "python", "react@0.14", "README", "vue@1.0.28"}
	readFromDir("devhints", blacklist)
	readFromDir("other", []string{"101v2"})

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
		cs.pathHTML = cs.fileNameBase + ".html"
		cs.htmlFullPath = filepath.Join(csDir, cs.pathHTML)
	}

	logf("%d cheatsheets\n", len(cheatsheets))

	nThreads := runtime.NumCPU()
	//nThreads = 1
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
