// This code is under BSD license. See license-bsd.txt
package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"time"
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

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func logIfError(err error) {
	if err != nil {
		logf("%s\n", err)
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

// can be used for http.Get() requests with better timeouts. New one must be created
// for each Get() request
func newTimeoutClient(connectTimeout time.Duration, readWriteTimeout time.Duration) *http.Client {
	timeoutDialer := func(cTimeout time.Duration, rwTimeout time.Duration) func(net, addr string) (c net.Conn, err error) {
		return func(netw, addr string) (net.Conn, error) {
			conn, err := net.DialTimeout(netw, addr, cTimeout)
			if err != nil {
				return nil, err
			}
			conn.SetDeadline(time.Now().Add(rwTimeout))
			return conn, nil
		}
	}

	return &http.Client{
		Transport: &http.Transport{
			Dial:  timeoutDialer(connectTimeout, readWriteTimeout),
			Proxy: http.ProxyFromEnvironment,
		},
	}
}
func httpPost(uri string, body []byte) ([]byte, error) {
	// default timeout for http.Get() is really long, so dial it down
	// for both connection and read/write timeouts
	timeoutClient := newTimeoutClient(time.Second*120, time.Second*120)
	resp, err := timeoutClient.Post(uri, "", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("'%s': status code not 200 (%d)", uri, resp.StatusCode)
	}
	return ioutil.ReadAll(resp.Body)
}
