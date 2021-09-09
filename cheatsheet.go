package main

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/kjk/u"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

type cheatsheet struct {
	path string
	name string // unique name from file name, without
}

func csMdToHTML(d []byte) []byte {
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
		),
	)
	var buf bytes.Buffer
	err := md.Convert(d, &buf)
	must(err)
	return buf.Bytes()
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
			d := u.ReadFileMust(cs.path)
			html := csMdToHTML(d)
			logf("Processed %s, html size: %d\n", cs.path, len(html))
			wg.Done()
			<-sem
		}(cs)
	}
	wg.Wait()
}
