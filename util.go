// This code is under BSD license. See license-bsd.txt
package main

import (
	"bytes"
	"context"
	"io/fs"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kjk/common/u"
)

var (
	must              = u.Must
	panicIf           = u.PanicIf
	panicIfErr        = u.PanicIfErr
	fileExists        = u.FileExists
	pathExists        = u.PathExists
	dirExists         = u.DirExists
	getFileSize       = u.FileSize
	perc              = u.Percent
	formatSize        = u.FormatSize
	isWindows         = u.IsWindows
	normalizeNewlines = u.NormalizeNewlines
	openBrowser       = u.OpenBrowser
	capitalize        = u.Capitalize
	copyFile          = u.CopyFile
)

func ctx() context.Context {
	return context.Background()
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

const base64Chars = "0123456789abcdefghijklmnopqrstuvwxyz"

// encodeBase64 encodes n as base64
func encodeBase64(n int) string {
	var buf [16]byte
	size := 0
	for {
		buf[size] = base64Chars[n%36]
		size++
		if n < 36 {
			break
		}
		n /= 36
	}
	end := size - 1
	for i := 0; i < end; i++ {
		b := buf[i]
		buf[i] = buf[end]
		buf[end] = b
		end--
	}
	return string(buf[:size])
}

func currDirAbsMust() string {
	dir, err := filepath.Abs(".")
	must(err)
	return dir
}

// we are executed for do/ directory so top dir is parent dir
func cdUpDir(dirName string) {
	startDir := currDirAbsMust()
	dir := startDir
	for {
		// we're already in top directory
		if filepath.Base(dir) == dirName && dirExists(dir) {
			err := os.Chdir(dir)
			must(err)
			return
		}
		parentDir := filepath.Dir(dir)
		panicIf(dir == parentDir, "invalid startDir: '%s', dir: '%s'", startDir, dir)
		dir = parentDir
	}
}

func fmtCmdShort(cmd exec.Cmd) string {
	cmd.Path = filepath.Base(cmd.Path)
	return cmd.String()
}

// RunCmdLoggedMust runs a command and returns its stdout
// Shows output as it happens
func runCmdLoggedMust(cmd *exec.Cmd) string {
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return runCmdMust(cmd)
}

func runCmdMust(cmd *exec.Cmd) string {
	logf(ctx(), "> %s\n", fmtCmdShort(*cmd))
	canCapture := (cmd.Stdout == nil) && (cmd.Stderr == nil)
	if canCapture {
		out, err := cmd.CombinedOutput()
		if err == nil {
			if len(out) > 0 {
				logf(ctx(), "Output:\n%s\n", string(out))
			}
			return string(out)
		}
		logf(ctx(), "cmd '%s' failed with '%s'. Output:\n%s\n", cmd, err, string(out))
		must(err)
		return string(out)
	}
	err := cmd.Run()
	if err == nil {
		return ""
	}
	logf(ctx(), "cmd '%s' failed with '%s'\n", cmd, err)
	must(err)
	return ""
}
