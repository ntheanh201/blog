package main

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/kjk/common/u"
)

func optimizeImages(dirs ...string) {
	var (
		sem              = make(chan bool, runtime.NumCPU()+1)
		wg               sync.WaitGroup
		nProcessed       int
		nOptimized       int
		imgoptSizeBefore int64
		sizeAFter        int64
		imgoptMu         sync.Mutex
	)

	optimizeWithOptipng := func(path string) {
		logf(ctx(), "Optimizing '%s'\n", path)
		sizeBefore := u.FileSize(path)
		cmd := exec.Command("optipng", "-o5", path)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			// it's ok if fails. some jpeg images are saved as .png
			// which trips it
			logf(ctx(), "optipng failed with '%s'\n", err)
		}
		sizeAfter := u.FileSize(path)
		panicIf(sizeBefore == -1 || sizeAfter == -1)

		imgoptMu.Lock()
		defer imgoptMu.Unlock()
		nProcessed++
		if sizeBefore != sizeAfter {
			nOptimized++
		}
		sizeAFter += sizeAfter
		imgoptSizeBefore += sizeBefore
	}

	maybeOptimizeImage := func(path string) {
		ext := filepath.Ext(path)
		ext = strings.ToLower(ext)
		switch ext {
		// TODO: for .gif requires -snip
		case ".png", ".tiff", ".tif", "bmp":
			wg.Add(1)
			// run optipng in parallel
			go func() {
				sem <- true
				optimizeWithOptipng(path)
				<-sem
				wg.Done()
			}()
		}
	}

	// verify we have optipng installed
	cmd := exec.Command("optipng", "-h")
	err := cmd.Run()
	panicIf(err != nil, "optipng is not installed")

	for _, dir := range dirs {
		filepath.WalkDir(dir, func(path string, e fs.DirEntry, err error) error {
			if err == nil && e.Type().IsRegular() {
				maybeOptimizeImage(path)
			}
			return nil
		})
	}
	wg.Wait()
	logf(ctx(), "optimizeAllImages: processed %d, optimized %d, %s => %s\n", nProcessed, nOptimized, formatSize(imgoptSizeBefore), formatSize(sizeAFter))
}
