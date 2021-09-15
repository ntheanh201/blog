---
title: Go Snippets
category: Go
---

# Intro

A collection of Go code snippets that I use often in my programs.

If function name ends with `Must`, it will panic on error.
This is ok for short scripts but not for long-running programs.

# Strings

## normalizeNewLines

```go
func normalizeNewlines(d []byte) []byte {
	// replace CR LF (windows) with LF (unix)
	d = bytes.Replace(d, []byte{13, 10}, []byte{10}, -1)
	// replace CF (mac) with LF (unix)
	d = bytes.Replace(d, []byte{13}, []byte{10}, -1)
	return d
}
```


## bytesRemoveFirstLine

```go
// return first line of d and the rest
func bytesRemoveFirstLine(d []byte) (string, []byte) {
	idx := bytes.IndexByte(d, 10)
	if -1 == idx {
		return string(d), nil
	}
	l := d[:idx]
	return string(l), d[idx+1:]
}
```

## sliceRmoveDuplicateStrings

```go
// sliceRemoveDuplicateStrings removes duplicate strings from an array of strings.
// It's optimized for the case of no duplicates. It modifes a in place.
func sliceRemoveDuplicateStrings(a []string) []string {
    if len(a) < 2 {
        return a
    }
    sort.Strings(a)
    writeIdx := 1
    for i := 1; i < len(a); i++ {
        if a[i-1] == a[i] {
            continue
        }
        if writeIdx != i {
            a[writeIdx] = a[i]
        }
        writeIdx++
    }
    return a[:writeIdx]
}
```

## stringInSlice

```go
func stringInSlice(a []string, toCheck string) bool {
	for _, s := range a {
		if s == toCheck {
			return true
		}
	}
	return false
}
```

# Files

## pathExists

```go
func pathExists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}
```

## fileExists

```go
func dirExists(path string) bool {
	st, err := os.Lstat(path)
	return err == nil && !st.IsDir() && st.IsRegular()
}
```

## dirExists

```go
func dirExists(path string) bool {
	st, err := os.Lstat(path)
	return err == nil && st.IsDir()
}
```

## getFileSize

```go
func getFileSize(path string) int64 {
	st, err := os.Lstat(path)
	if err == nil {
		return st.Size()
	}
	return -1
}
```

## copyFile

Copy file, ensures to create a destination directory.

```go
func copyFile(dst string, src string) error {
	err := os.MkdirAll(filepath.Dir(dst), 0755)
	if err != nil {
		return err
	}
	fin, err := os.Open(src)
	if err != nil {
		return err
	}
	defer fin.Close()
	fout, err := os.Create(dst)
	if err != nil {
		return err
	}

	_, err = io.Copy(fout, fin)
	err2 := fout.Close()
	if err != nil || err2 != nil {
		os.Remove(dst)
	}

	return err
}
```

## createDirForFile

```go
func createDirForFile(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0755)
}
```

## readGzippedFile

```go
func readGzippedFile(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	gr, err := gzip.NewReader(file)
	if err != nil {
		return nil, err
	}
	defer gr.Close()
	return ioutil.ReadAll(gr)
}
```

## readFileLines

```go
func readFileLines(filePath string) ([]string, error) {
    file, err := os.OpenFile(filePath, os.O_RDONLY, 0666)
    if err != nil {
        return nil, err
    }
    defer file.Close()
    scanner := bufio.NewScanner(file)
    res := make([]string, 0)
    for scanner.Scan() {
        line := scanner.Bytes()
        res = append(res, string(line))
    }
    if err = scanner.Err(); err != nil {
        return nil, err
    }
    return res, nil
}
```

## sha1HexOfFile

```go
func sha1OfFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		//fmt.Printf("os.Open(%s) failed with %s\n", path, err.Error())
		return nil, err
	}
	defer f.Close()
	h := sha1.New()
	_, err = io.Copy(h, f)
	if err != nil {
		//fmt.Printf("io.Copy() failed with %s\n", err.Error())
		return nil, err
	}
	return h.Sum(nil), nil
}

func sha1HexOfFile(path string) (string, error) {
	sha1, err := sha1OfFile(path)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha1), nil
}
```

## unzipToDir

```go
func recreateDir(dir string) error {
	err := os.RemoveAll(dir)
	if err != nil {
		return err
	}
	return os.MkdirAll(dir, 0755)
}

func createDirForFile(path string) error {
	dir := filepath.Dir(path)
	return os.MkdirAll(dir, 0755)
}

func unzipFile(f *zip.File, dstPath string) error {
	r, err := f.Open()
	if err != nil {
		return err
	}
	defer r.Close()

	err = createDirForFile(dstPath)
	if err != nil {
		return err
	}

	w, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, r)
	if err != nil {
		w.Close()
		os.Remove(dstPath)
		return err
	}
	err = w.Close()
	if err != nil {
		os.Remove(dstPath)
		return err
	}
	return nil
}

func unzipToDir(zipPath string, destDir string) error {
	st, err := os.Stat(zipPath)
	if err != nil {
		return err
	}
	fileSize := st.Size()
	f, err := os.Open(zipPath)
	if err != nil {
		return err
	}
	defer f.Close()

	zr, err := zip.NewReader(f, fileSize)
	if err != nil {
		return err
	}
	err = recreateDir(destDir)
	if err != nil {
		return err
	}

	for _, fi := range zr.File {
		if fi.FileInfo().IsDir() {
			continue
		}
		destPath := filepath.Join(destDir, fi.Name)
		err = unzipFile(fi, destPath)
		if err != nil {
			os.RemoveAll(destDir)
			return err
		}
	}
	return nil
}
```

# HTTP

## httpGet

```go
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

func httpGet(url string) ([]byte, error) {
    // default timeout for http.Get() is really long, so dial it down
    // for both connection and read/write timeouts
    timeoutClient := newTimeoutClient(time.Second*120, time.Second*120)
    resp, err := timeoutClient.Get(url)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    if resp.StatusCode != 200 {
        return nil, errors.New(fmt.Sprintf("'%s': status code not 200 (%d)", url, resp.StatusCode))
    }
    return ioutil.ReadAll(resp.Body)
}
```

## httpPost

```go
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
```

## httpPostMultiPart

```go
func httpPostMultiPart(uri string, files map[string]string) ([]byte, error) {
	contentType, body, err := createMultiPartForm(files)
	if err != nil {
		return nil, err
	}
	// default timeout for http.Get() is really long, so dial it down
	// for both connection and read/write timeouts
	timeoutClient := newTimeoutClient(time.Second*120, time.Second*120)
	resp, err := timeoutClient.Post(uri, contentType, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("'%s': status code not 200 (%d)", uri, resp.StatusCode)
	}
	return ioutil.ReadAll(resp.Body)
}

func createMultiPartForm(form map[string]string) (string, io.Reader, error) {
	body := new(bytes.Buffer)
	mp := multipart.NewWriter(body)
	defer mp.Close()
	for key, val := range form {
		if strings.HasPrefix(val, "@") {
			val = val[1:]
			file, err := os.Open(val)
			if err != nil {
				return "", nil, err
			}
			defer file.Close()
			part, err := mp.CreateFormFile(key, val)
			if err != nil {
				return "", nil, err
			}
			io.Copy(part, file)
		} else {
			mp.WriteField(key, val)
		}
	}
	return mp.FormDataContentType(), body, nil
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
```


# Misc

## must

```go
func must(err error) {
	if err != nil {
		panic(err.Error())
	}
}
```

## panicIf

```go
func panicIf(cond bool, arg ...interface{}) {
	if !cond {
		return
	}
	s := "condition failed"
	if len(arg) > 0 {
		s = fmt.Sprintf("%s", arg[0])
		if len(arg) > 1 {
			s = fmt.Sprintf(s, arg[1:]...)
		}
	}
	panic(s)
}
```

## logf

```go
func logf(s string, arg ...interface{}) {
	if len(arg) > 0 {
		s = fmt.Sprintf(s, arg...)
	}
	fmt.Print(s)
}
```

## logIfErr

```go
func logIfErr(err error) {
	if err != nil {
		logf(err.Error())
	}
}
```

## isWindows

```go
func isWindows() bool {
	return strings.Contains(runtime.GOOS, "windows")
}
```

## userHomeDirMust

```go
func userHomeDirMust() string {
	s, err := os.UserHomeDir()
	must(err)
	return s
}
```

## non-blocking channel send

```go
// if ch if full we will not block, thanks to default case
select {
case ch <- value:
default:
}
```

## openBrowser

```go
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
```

## ProgressEstimator

```go
package util

import (
	"sync"
	"time"
)

// ProgressEstimatorData contains fields readable after Next()
type ProgressEstimatorData struct {
	Total             int
	Curr              int
	Left              int
	PercDone          float64 // 0...1
	Skipped           int
	TimeSoFar         time.Duration
	EstimatedTimeLeft time.Duration
}

// ProgressEstimator is for estimating progress
type ProgressEstimator struct {
	timeStart time.Time
	sync.Mutex
	ProgressEstimatorData
}

// NewProgressEstimator creates a ProgressEstimator
func NewProgressEstimator(total int) *ProgressEstimator {
	d := ProgressEstimatorData{
		Total: total,
	}
	return &ProgressEstimator{
		ProgressEstimatorData: d,
		timeStart:             time.Now(),
	}
}

// Next advances estimator
func (pe *ProgressEstimator) next(isSkipped bool) ProgressEstimatorData {
	pe.Lock()
	defer pe.Unlock()
	if isSkipped {
		pe.Skipped++
	}
	pe.Curr++
	pe.Left = pe.Total - pe.Curr
	pe.TimeSoFar = time.Since(pe.timeStart)

	realTotal := pe.Total - pe.Skipped
	realCurr := pe.Curr - pe.Skipped
	if realCurr == 0 || realTotal == 0 {
		pe.EstimatedTimeLeft = pe.TimeSoFar
	} else {
		pe.PercDone = float64(realCurr) / float64(realTotal) // 0..1 range
		realPerc := float64(realTotal) / float64(realCurr)
		estimatedTotalTime := float64(pe.TimeSoFar) * realPerc
		pe.EstimatedTimeLeft = time.Duration(estimatedTotalTime) - pe.TimeSoFar
	}
	cpy := pe.ProgressEstimatorData
	return cpy
}

// Next advances estimator
func (pe *ProgressEstimator) Next() ProgressEstimatorData {
	return pe.next(false)
}

// Skip advances estimator but allows to mark this file as taking no time,
// to allow better estimates
func (pe *ProgressEstimator) Skip() ProgressEstimatorData {
	return pe.next(true)
}
```

## Debouncer

```go
// Debouncer runs a given function, debounced
type Debouncer struct {
    currDebounceID *int32
}

// NewDebouncer creates a new Debouncer
func NewDebouncer() *Debouncer {
    return &Debouncer{}
}

func (d *Debouncer) debounce(f func(), timeout time.Duration) {
    if d.currDebounceID != nil {
        // stop currently scheduled function
        v := atomic.AddInt32(d.currDebounceID, 1)
        d.currDebounceID = nil
        if v > 1 {
            // it was already executed
            return
        }
    }

    d.currDebounceID = new(int32)
    go func(f func(), timeout time.Duration, debounceID *int32) {
        for {
            select {
            case <-time.After(timeout):
                v := atomic.AddInt32(debounceID, 1)
                // if v != 1, it was cancelled
                if v == 1 {
                    f()
                }
            }
        }
    }(f, timeout, d.currDebounceID)
}

var articleLoadDebouncer *Debouncer

func reloadArticlesDelayed() {
    if articleLoadDebouncer == nil {
        articleLoadDebouncer = NewDebouncer()
    }
    articleLoadDebouncer.debounce(loadArticles, time.Second)
}
```

## mimeTypeFromFileName

```go
func mimeTypeFromFileName(path string) string {
	var mimeTypes = map[string]string{
		// this is a list from go's mime package
		".css":  "text/css; charset=utf-8",
		".gif":  "image/gif",
		".htm":  "text/html; charset=utf-8",
		".html": "text/html; charset=utf-8",
		".jpg":  "image/jpeg",
		".js":   "application/javascript",
		".wasm": "application/wasm",
		".pdf":  "application/pdf",
		".png":  "image/png",
		".svg":  "image/svg+xml",
		".xml":  "text/xml; charset=utf-8",

		// those are my additions
		".txt":  "text/plain",
		".exe":  "application/octet-stream",
		".json": "application/json",
	}

	ext := strings.ToLower(filepath.Ext(path))
	mt := mimeTypes[ext]
	if mt != "" {
		return mt
	}
	// if not given, default to this
	return "application/octet-stream"
}
```

## formatSize

```go
func formatSize(n int64) string {
	sizes := []int64{1024*1024*1024, 1024*1024, 1024}
	suffixes := []string{"GB", "MB", "kB"}

	for i, size := range sizes {
		if n >= size {
			s := fmt.Sprintf("%.2f", float64(n)/float64(size))
			return strings.TrimSuffix(s, ".00") + " " + suffixes[i]
		}
	}
	return fmt.Sprintf("%d bytes", i)
}
```

## formatDuration

```go
// time.Duration with a better string representation
type FormattedDuration time.Duration

func (d FormattedDuration) String() string {
	return formatDuration(time.Duration(d))
}

// formats duration in a more human friendly way
// than time.Duration.String()
func formatDuration(d time.Duration) string {
	s := d.String()
	if strings.HasSuffix(s, "µs") {
		// for µs we don't want fractions
		parts := strings.Split(s, ".")
		if len(parts) > 1 {
			return parts[0] + " µs"
		}
		return strings.ReplaceAll(s, "µs", " µs")
	} else if strings.HasSuffix(s, "ms") {
		// for ms we only want 2 digit fractions
		parts := strings.Split(s, ".")
		//fmt.Printf("fmtDur: '%s' => %#v\n", s, parts)
		if len(parts) > 1 {
			s2 := parts[1]
			if len(s2) > 4 {
				// 2 for "ms" and 2+ for fraction
				res := parts[0] + "." + s2[:2] + " ms"
				//fmt.Printf("fmtDur: s2: '%s', res: '%s'\n", s2, res)
				return res
			}
		}
		return strings.ReplaceAll(s, "ms", " ms")
	}
	return s
}
```


## expandTildeInPath

Given "~/foo", will replace "~" with home directory.

```go
func expandTildeInPath(s string) string {
	if strings.HasPrefix(s, "~") {
		dir, err := os.UserHomeDir()
		must(err)
		return dir + s[1:]
	}
	return s
}
```


## runCmdMust

```go
func fmtCmdShort(cmd exec.Cmd) string {
	cmd.Path = filepath.Base(cmd.Path)
	return cmd.String()
}

func runCmdMust(cmd *exec.Cmd) string {
	logf("> %s\n", fmtCmdShort(*cmd))
	canCapture := (cmd.Stdout == nil) && (cmd.Stderr == nil)
	if canCapture {
		out, err := cmd.CombinedOutput()
		if err == nil {
			if len(out) > 0 {
				logf("Output:\n%s\n", string(out))
			}
			return string(out)
		}
		logf("cmd '%s' failed with '%s'. Output:\n%s\n", cmd, err, string(out))
		must(err)
		return string(out)
	}
	err := cmd.Run()
	if err == nil {
		return ""
	}
	logf("cmd '%s' failed with '%s'\n", cmd, err)
	must(err)
	return ""
}
```

## runCmdLogged

```go
func runCmdLogged(cmd *exec.Cmd) error {
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
```

## cdUpDir

```go
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
```
