// This code is under BSD license. See license-bsd.txt
package main

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func panicIf(cond bool, args ...interface{}) {
	if !cond {
		return
	}
	s := "condition failed"
	if len(args) > 0 {
		s = fmt.Sprintf("%s", args[0])
		if len(args) > 1 {
			s = fmt.Sprintf(s, args[1:]...)
		}
	}
	panic(s)
}

func panicIfErr(err error, args ...interface{}) {
	if err == nil {
		return
	}
	s := err.Error()
	if len(args) > 0 {
		s = fmt.Sprintf("%s", args[0])
		if len(args) > 1 {
			s = fmt.Sprintf(s, args[1:]...)
		}
	}
	panic(s)
}

func ctx() context.Context {
	return context.Background()
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func logIfError(err error) {
	if err != nil {
		logf(ctx(), "%s\n", err)
	}
}

// whitelisted characters valid in url
func validateRune(c rune) byte {
	if c >= 'a' && c <= 'z' {
		return byte(c)
	}
	if c >= '0' && c <= '9' {
		return byte(c)
	}
	if c == '-' || c == '_' || c == '.' {
		return byte(c)
	}
	if c == ' ' {
		return '-'
	}
	return 0
}

func charCanRepeat(c byte) bool {
	if c >= 'a' && c <= 'z' {
		return true
	}
	if c >= '0' && c <= '9' {
		return true
	}
	return false
}

// urlify generates safe url from tile by removing hazardous characters
func urlify(title string) string {
	s := strings.TrimSpace(title)
	s = strings.ToLower(s)
	var res []byte
	for _, r := range s {
		c := validateRune(r)
		if c == 0 {
			continue
		}
		// eliminute duplicate consequitive characters
		var prev byte
		if len(res) > 0 {
			prev = res[len(res)-1]
		}
		if c == prev && !charCanRepeat(c) {
			continue
		}
		res = append(res, c)
	}
	s = string(res)
	if len(s) > 128 {
		s = s[:128]
	}
	return s
}

func trimEmptyLines(a []string) []string {
	var res []string

	// remove empty lines from beginning and duplicated empty lines
	prevWasEmpty := true
	for _, s := range a {
		currIsEmpty := (len(s) == 0)
		if currIsEmpty && prevWasEmpty {
			continue
		}
		res = append(res, s)
		prevWasEmpty = currIsEmpty
	}
	// remove empty lines from end
	for len(res) > 0 {
		lastIdx := len(res) - 1
		if len(res[lastIdx]) != 0 {
			break
		}
		res = res[:lastIdx]
	}
	return res
}

func findWordEnd(s string, start int) int {
	for i := start; i < len(s); i++ {
		c := s[i]
		if c == ' ' {
			return i + 1
		}
	}
	return -1
}

// remove #tag from start and end
func removeHashTags(s string) (string, []string) {
	var tags []string
	defer func() {
		for i, tag := range tags {
			tags[i] = strings.ToLower(tag)
		}
	}()

	// remove hashtags from start
	for strings.HasPrefix(s, "#") {
		idx := findWordEnd(s, 0)
		if idx == -1 {
			tags = append(tags, s[1:])
			return "", tags
		}
		tags = append(tags, s[1:idx-1])
		s = strings.TrimLeft(s[idx:], " ")
	}

	// remove hashtags from end
	s = strings.TrimRight(s, " ")
	for {
		idx := strings.LastIndex(s, "#")
		if idx == -1 {
			return s, tags
		}
		// tag from the end must not have space after it
		if -1 != findWordEnd(s, idx) {
			return s, tags
		}
		// tag from the end must start at the beginning of line
		// or be proceded by space
		if idx > 0 && s[idx-1] != ' ' {
			return s, tags
		}
		tags = append(tags, s[idx+1:])
		s = strings.TrimRight(s[:idx], " ")
	}
}

func replaceExt(fileName, newExt string) string {
	ext := filepath.Ext(fileName)
	if ext == "" {
		return fileName
	}
	n := len(fileName)
	s := fileName[:n-len(ext)]
	return s + newExt
}

// foo => Foo, BAR => Bar etc.
func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	s = strings.ToLower(s)
	return strings.ToUpper(s[0:1]) + s[1:]
}

func normalizeNewlines(d []byte) []byte {
	// replace CR LF (windows) with LF (unix)
	d = bytes.Replace(d, []byte{13, 10}, []byte{10}, -1)
	// replace CF (mac) with LF (unix)
	d = bytes.Replace(d, []byte{13}, []byte{10}, -1)
	return d
}

func toTrimmedLines(d []byte) []string {
	lines := strings.Split(string(d), "\n")
	i := 0
	for _, l := range lines {
		l = strings.TrimSpace(l)
		// remove empty lines
		if len(l) > 0 {
			lines[i] = l
			i++
		}
	}
	return lines[:i]
}

func readFileMust(path string) []byte {
	d, err := ioutil.ReadFile(path)
	must(err)
	return d
}

// from https://gist.github.com/hyg/9c4afcd91fe24316cbf0
func openBrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Fatal(err)
	}
}

func formatSize(n int64) string {
	sizes := []int64{1024 * 1024 * 1024, 1024 * 1024, 1024}
	suffixes := []string{"GB", "MB", "kB"}
	for i, size := range sizes {
		if n >= size {
			s := fmt.Sprintf("%.2f", float64(n)/float64(size))
			return strings.TrimSuffix(s, ".00") + " " + suffixes[i]
		}
	}
	return fmt.Sprintf("%d bytes", n)
}

func dirToLF(dir string) {
	filepath.WalkDir(dir, func(path string, e fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if e.IsDir() || !e.Type().IsRegular() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		shouldProcess := false
		switch ext {
		case ".js", ".css", ".html", ".md", ".txt", ".go":
			shouldProcess = true
		}
		if !shouldProcess {
			return nil
		}
		d := readFileMust(path)
		d2 := normalizeNewlines(d)
		if !bytes.Equal(d, d2) {
			logf(ctx(), path+"\n")
			must(ioutil.WriteFile(path, d2, 0644))
		}
		return nil
	})
}

func createDirForFile(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0755)
}

func fileExists(path string) bool {
	st, err := os.Lstat(path)
	return err == nil && st.Mode().IsRegular()
}
