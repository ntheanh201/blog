package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/gomarkdown/markdown"
	mdhtml "github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

func csBuildToc(parser *parser.Parser, md []byte) {
	doc := parser.Parse(md)
	logf("%#v\n", doc)
}

// TODO: more work needed:
// - parse YAML metadata and remove from markdown
// - generate toc
func csGenHTML(cs *cheatSheet) {
	extensions := parser.NoIntraEmphasis |
		parser.Tables |
		parser.FencedCode |
		parser.Autolink |
		parser.Strikethrough |
		parser.SpaceHeadings |
		parser.NoEmptyLineBeforeBlock
	parser := parser.NewWithExtensions(extensions)

	csBuildToc(parser, cs.md)

	htmlFlags := mdhtml.Smartypants |
		mdhtml.SmartypantsFractions |
		mdhtml.SmartypantsDashes |
		mdhtml.SmartypantsLatexDashes
	htmlOpts := mdhtml.RendererOptions{
		Flags:          htmlFlags,
		RenderNodeHook: makeRenderHookCodeBlock(""),
	}
	renderer := mdhtml.NewRenderer(htmlOpts)
	cs.html = markdown.ToHTML(cs.md, parser, renderer)
	logf("Processed %s, html size: %d\n", cs.mdPath, len(cs.html))
}

type cheatSheet struct {
	mdPath     string
	name       string // unique name from file name, without extension
	htmlPath   string
	mdWithMeta []byte
	md         []byte
	meta       map[string]string
	html       []byte
}

func extractCheatSheetMetadata(cs *cheatSheet) {

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
			path := filepath.Join(dir, name)
			name = strings.Split(name, ".")[0]
			cs := &cheatSheet{
				mdPath: path,
				name:   name,
				meta:   map[string]string{},
			}
			cheatsheets = append(cheatsheets, cs)
		}
	}

	dir := filepath.Join("www", "cheatsheets", "devhints")
	readFromDir(dir)
	dir = filepath.Join("www", "cheatsheets", "other")
	readFromDir(dir)

	// TODO: uniquify names
	for _, cs := range cheatsheets {
		cs.htmlPath = filepath.Join("www", "cheatsheets", cs.name+".html")
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
}
