package main

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

var (
	imgoptSem        chan bool
	imgoptWg         sync.WaitGroup
	imgoptProcessed  int
	imgoptOptimized  int
	imgoptSizeBefore int64
	imgoptSizeAFter  int64
	imgoptMu         sync.Mutex
)

// run optipng in parallel
func optimizeWithOptipng(path string) {
	logf(ctx(), "Optimizing '%s'\n", path)
	sizeBefore := getFileSize(path)
	cmd := exec.Command("optipng", "-o5", path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		// it's ok if fails. some jpeg images are saved as .png
		// which trips it
		logf(ctx(), "optipng failed with '%s'\n", err)
		return
	}
	sizeAfter := getFileSize(path)
	panicIf(sizeBefore == -1 || sizeAfter == -1)

	imgoptMu.Lock()
	defer imgoptMu.Unlock()
	imgoptProcessed++
	if sizeBefore != sizeAfter {
		imgoptOptimized++
	}
	imgoptSizeAFter += sizeAfter
	imgoptSizeBefore += sizeBefore
}

func maybeOptimizeImage(path string) {
	ext := filepath.Ext(path)
	ext = strings.ToLower(ext)
	switch ext {
	// TODO: for .gif requires -snip
	case ".png", ".tiff", ".tif", "bmp":
		imgoptWg.Add(1)
		go func() {
			imgoptSem <- true
			optimizeWithOptipng(path)
			<-imgoptSem
			imgoptWg.Done()
		}()
	}
}

func optimizeAllImages(dirsToVisit []string) {
	imgoptSem = make(chan bool, runtime.NumCPU()+1)

	timeStart := time.Now()
	defer func() {
		logf(ctx(), "optimizeAllImages: took %s\n", time.Since(timeStart))
	}()

	// verify we have optipng installed
	cmd := exec.Command("optipng", "-h")
	err := cmd.Run()
	panicIf(err != nil, "optipng is not installed")

	for len(dirsToVisit) > 0 {
		dir := dirsToVisit[0]
		dirsToVisit = dirsToVisit[1:]
		files, err := ioutil.ReadDir(dir)
		must(err)
		for _, f := range files {
			name := f.Name()
			path := filepath.Join(dir, name)
			if f.IsDir() {
				if name == "www" || name == "covers" || name == "tmpl" {
					// assume already optimized
				} else {
					dirsToVisit = append(dirsToVisit, path)
				}
				continue
			}
			maybeOptimizeImage(path)
		}
	}
	imgoptWg.Wait()
	logf(ctx(), "optimizeAllImages: processed %d, optimized %d, %s => %s\n", imgoptProcessed, imgoptOptimized, formatSize(imgoptSizeBefore), formatSize(imgoptSizeAFter))
}
