package main

import (
	"fmt"
	"os"
	"path/filepath"
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

func makeUniqueID(taken map[string]bool, id string) string {
	curr := id
	n := 0
	for taken[curr] {
		n++
		curr = fmt.Sprintf("%s%d", id, n)
	}
	taken[curr] = true
	return curr
}

type tocNode struct {
	heading   *ast.Heading
	children  []*ast.Heading
	nSiblings int
}

func csBuildToc(md []byte) {
	logf("csBuildToc: printing heading, len(md): %d\n", len(md))
	parser := newCsMarkdownParser()
	doc := parser.Parse(md)
	//ast.Print(os.Stdout, doc)

	var currHeading *ast.Heading
	var currHeadingContent string
	taken := map[string]bool{}
	ast.WalkFunc(doc, func(node ast.Node, entering bool) ast.WalkStatus {
		switch v := node.(type) {
		case *ast.Heading:
			if entering {
				currHeading = v
			} else {
				currHeading.HeadingID = makeUniqueID(taken, currHeading.HeadingID)
				logf("h%d #%s %s\n", currHeading.Level, currHeading.HeadingID, currHeadingContent)
				currHeading = nil
				currHeadingContent = ""
			}
		case *ast.Text:
			// the only child of ast.Heading is ast.Text (I think)
			if currHeading != nil {
				currHeadingContent = string(v.Literal)
			}
		}
		return ast.GoToNext
	})
}

func csGenHTML(cs *cheatSheet) {
	logf("csGenHTML: for '%s'\n", cs.mdPath)
	csBuildToc(cs.md)
	parser := newMarkdownParser()
	renderer := newMarkdownHTMLRenderer("")
	cs.html = markdown.ToHTML(cs.md, parser, renderer)
	logf("Processed %s, html size: %d\n", cs.mdPath, len(cs.html))
}

type cheatSheet struct {
	fileNameBase string // unique name from file name, without extension
	mdPath       string
	htmlPath     string
	mdWithMeta   []byte
	md           []byte
	meta         map[string]string
	html         []byte
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

func cheatsheets() {
	cheatsheets := []*cheatSheet{}

	readFromDir := func(dir string) {
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
			mdPath := filepath.Join(dir, name)
			name = strings.Split(name, ".")[0]
			if name != "go" {
				//logf("%s ", name)
				continue
			}
			logf("%s\n", mdPath)
			cs := &cheatSheet{
				fileNameBase: name,
				mdPath:       mdPath,
				meta:         map[string]string{},
			}
			cheatsheets = append(cheatsheets, cs)
		}
	}

	dir := filepath.Join("www", "cheatsheets", "devhints")
	readFromDir(dir)
	dir = filepath.Join("www", "cheatsheets", "other")
	readFromDir(dir)

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
		cs.htmlPath = filepath.Join("www", "cheatsheets", cs.fileNameBase+".html")
	}

	logf("%d cheatsheets\n", len(cheatsheets))

	//sem := make(chan bool, runtime.NumCPU())
	sem := make(chan bool, 1)
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
		name := filepath.Base(cs.htmlPath)
		files[name] = d
	}
	uri := uploadFilesToInstantPreviewMust(files)
	logf("uploaded %d cheatsheets under: %s\n", len(cheatsheets), uri)
}
