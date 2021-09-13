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

type cheatsheet struct {
	path string
	name string // unique name from file name, without
}

func csMdToHTML(md []byte, defaultLang string) []byte {
	extensions := parser.NoIntraEmphasis |
		parser.Tables |
		parser.FencedCode |
		parser.Autolink |
		parser.Strikethrough |
		parser.SpaceHeadings |
		parser.NoEmptyLineBeforeBlock
	parser := parser.NewWithExtensions(extensions)

	htmlFlags := mdhtml.Smartypants |
		mdhtml.SmartypantsFractions |
		mdhtml.SmartypantsDashes |
		mdhtml.SmartypantsLatexDashes
	htmlOpts := mdhtml.RendererOptions{
		Flags:          htmlFlags,
		RenderNodeHook: makeRenderHookCodeBlock(defaultLang),
	}
	renderer := mdhtml.NewRenderer(htmlOpts)
	return markdown.ToHTML(md, parser, renderer)
}

func cheatsheets() {
	cheatsheets := []*cheatsheet{}
	{
		dir := filepath.Join("www", "cheatsheets", "devhints")
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
			cs := &cheatsheet{
				path: path,
				name: name,
			}
			cheatsheets = append(cheatsheets, cs)
		}
	}

	{
		dir := filepath.Join("www", "cheatsheets", "other")
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
			cs := &cheatsheet{
				path: path,
				name: name,
			}
			cheatsheets = append(cheatsheets, cs)
		}
	}

	// TODO: uniquify names
	logf("%d cheatsheets\n", len(cheatsheets))
	sem := make(chan bool, runtime.NumCPU())
	var wg sync.WaitGroup
	for _, cs := range cheatsheets {
		wg.Add(1)
		sem <- true
		go func(cs *cheatsheet) {
			d := readFileMust(cs.path)
			html := csMdToHTML(d, "")
			logf("Processed %s, html size: %d\n", cs.path, len(html))
			wg.Done()
			<-sem
		}(cs)
	}
	wg.Wait()
}
