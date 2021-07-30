package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	_ "net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/kjk/notionapi"
	"github.com/kjk/siser"
	"github.com/kjk/u"
)

var (
	must       = u.Must
	fatalIf    = u.PanicIf
	panicIf    = u.PanicIf
	panicIfErr = u.PanicIfErr

	analyticsURL    = `` // empty to disable
	analytics404URL = `` // empty to disable
	//analyticsURL = `http://localhost:8333/a/a.js?localhost` // for local testing
	//analytics404URL = `http://localhost:8333/a/a.js?localhost&404` // for local testing
	//analyticsURL = `https://analytics-w5yuy.ondigitalocean.app/a/a.js?localhost`
	//analytics404URL = `https://analytics-w5yuy.ondigitalocean.app/a/a.js?localhost&404`
)

const (
	htmlDir = "www_generated" // directory where we generate html files
)

var (
	flgVerbose bool
)

type RequestCacheEntry struct {
	Method   string
	URL      string
	Body     []byte
	BodyPP   []byte // only if different than Body
	Response []byte
}

type Cache struct {
	Path    string
	Entries []*RequestCacheEntry
}

func analyticsHTML() template.HTML {
	if analyticsURL == "" {
		return template.HTML("")
	}
	html := `<script defer data-domain="blog.kowalczyk.info" src="` + analyticsURL + `"></script>`
	return template.HTML(html)
}

func analytics404HTML() template.HTML {
	if analytics404URL == "" {
		return template.HTML("")
	}
	html := `<script defer data-domain="blog.kowalczyk.info" src="` + analytics404URL + `"></script>`
	return template.HTML(html)
}

const (
	recCacheName = "httpcache-v1"
)

func recGetKey(r *siser.Record, key string, pErr *error) string {
	if *pErr != nil {
		return ""
	}
	v, ok := r.Get(key)
	if !ok {
		*pErr = fmt.Errorf("didn't find key '%s'", key)
	}
	return v
}

func recGetKeyBytes(r *siser.Record, key string, pErr *error) []byte {
	return []byte(recGetKey(r, key, pErr))
}

func deserializeCache(d []byte) (*Cache, error) {
	br := bufio.NewReader(bytes.NewBuffer(d))
	r := siser.NewReader(br)
	r.NoTimestamp = true
	var err error
	c := &Cache{}
	for r.ReadNextRecord() {
		if r.Name != recCacheName {
			return nil, fmt.Errorf("unexpected record type '%s', wanted '%s'", r.Name, recCacheName)
		}
		rr := &RequestCacheEntry{}
		rr.Method = recGetKey(r.Record, "Method", &err)
		rr.URL = recGetKey(r.Record, "URL", &err)
		rr.Body = recGetKeyBytes(r.Record, "Body", &err)
		rr.Response = recGetKeyBytes(r.Record, "Response", &err)
		c.Entries = append(c.Entries, rr)
	}
	if err != nil {
		return nil, err
	}
	return c, nil
}

func testLoadCache(dir string) {
	timeStart := time.Now()
	entries, err := ioutil.ReadDir(dir)
	must(err)
	nFiles := 0

	var caches []*Cache
	for _, fi := range entries {
		if !fi.Mode().IsRegular() {
			continue
		}
		name := fi.Name()
		if !strings.HasSuffix(name, ".txt") {
			continue
		}
		nFiles++
		path := filepath.Join(dir, name)
		d := u.ReadFileMust(path)
		c, err := deserializeCache(d)
		must(err)
		caches = append(caches, c)
	}
	fmt.Printf("testLoadCache() loaded %d files in %s, %d caches\n", nFiles, time.Since(timeStart), len(caches))
}

func rebuildAll(d *notionapi.CachingClient) *Articles {
	regenMd()
	loadTemplates()
	articles := loadArticles(d)
	readRedirects(articles)
	generateHTML(articles)
	return articles
}

// caddy -log stdout
func runCaddy() {
	cmd := exec.Command("caddy", "-log", "stdout")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

func runWranglerDev() {
	cmd := exec.Command("wrangler", "dev")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	logIfError(err)
}

/*
func stopCaddy(cmd *exec.Cmd) {
	cmd.Process.Kill()
}
*/

func preview() {
	go func() {
		time.Sleep(time.Second * 1)
		u.OpenBrowser("http://localhost:8787")
	}()
	runWranglerDev()
}

var (
	nDownloadedPage = 0
	nReadFromCache  = 0
)

func eventObserver(ev interface{}) {
	switch v := ev.(type) {
	case *notionapi.EventError:
		logf(v.Error)
	case *notionapi.EventDidDownload:
		nDownloadedPage++
		title := ""
		if v.Page != nil {
			title = shortenString(notionapi.TextSpansToString(v.Page.Root().GetTitle()), 32)
		}
		logf("%03d %s '%s' : downloaded in %s\n", nDownloadedPage, v.PageID, title, v.Duration)
	case *notionapi.EventDidReadFromCache:
		// TODO: only verbose
		nDownloadedPage++
		nReadFromCache++
		title := ""
		if v.Page != nil {
			title = shortenString(notionapi.TextSpansToString(v.Page.Root().GetTitle()), 32)
		}
		logvf("%03d %s %s : read from cache in %s\n", nDownloadedPage, v.PageID, title, v.Duration)
		if nReadFromCache < 2 {
			logf("%03d %s %s : read from cache in %s\n", nDownloadedPage, v.PageID, title, v.Duration)
		}
	case *notionapi.EventGotVersions:
		logf("downloaded info about %d versions in %s\n", v.Count, v.Duration)
	}
}

func newNotionClient() *notionapi.Client {
	token := os.Getenv("NOTION_TOKEN")
	if token == "" {
		logf("must set NOTION_TOKEN env variable\n")
		//flag.Usage()
		os.Exit(1)
	}
	// TODO: verify token still valid, somehow
	client := &notionapi.Client{
		AuthToken: token,
	}
	if flgVerbose {
		client.Logger = os.Stdout
	}
	return client
}

func recreateDir(dir string) {
	err := os.RemoveAll(dir)
	must(err)
	err = os.MkdirAll(dir, 0755)
	must(err)
}

func main() {
	var (
		flgDeployDev       bool
		flgDeployProd      bool
		flgPreview         bool
		flgPreviewOnDemand bool
		flgNoCache         bool
		flgWc              bool
		flgRedownload      bool
		flgRedownloadOne   string
		flgRebuildHTML     bool
		flgDiff            bool
		flgCiDaily         bool
		flgCiBuild         bool
	)

	{
		flag.BoolVar(&flgWc, "wc", false, "wc -l i.e. line count")
		flag.BoolVar(&flgVerbose, "verbose", false, "if true, verbose logging")
		flag.BoolVar(&flgNoCache, "no-cache", false, "if true, disables cache for downloading notion pages")
		flag.BoolVar(&flgDeployDev, "deploy-dev", false, "deploy to https://blog.kjk.workers.dev/")
		flag.BoolVar(&flgDeployProd, "deploy-prod", false, "deploy to https://blog.kowalczyk.info")
		flag.BoolVar(&flgPreview, "preview", false, "runs caddy and opens a browser for preview")
		flag.BoolVar(&flgPreviewOnDemand, "preview-on-demand", false, "runs the browser for local preview")
		flag.BoolVar(&flgRedownload, "redownload-notion", false, "re-download the content from Notion. use -no-cache to disable cache")
		flag.StringVar(&flgRedownloadOne, "redownload-one", "", "re-download a single Notion page. use -no-cache to disable cache")
		flag.BoolVar(&flgRebuildHTML, "rebuild-html", false, "rebuild html in www_generated/ directory")
		//flag.BoolVar(&flgDiff, "diff", false, "preview diff using winmerge")
		flag.BoolVar(&flgCiBuild, "ci-build", false, "runs on GitHub CI for every checkin")
		flag.BoolVar(&flgCiDaily, "ci-daily", false, "runs once a day on GitHub CI")
		flag.Parse()
	}

	timeStart := time.Now()
	defer func() {
		logf("finished in %s\n", time.Since(timeStart))
	}()

	if false {
		flgRedownloadOne = "08e19004306b413aba6e0e86a10fec7a"
	}

	if false {
		testLoadCache("notion_cache")
		return
	}

	openLog()
	defer closeLog()

	if flgWc {
		doLineCount()
		return
	}

	if flgDiff {
		winmergeDiffPreview()
		return
	}

	hasCmd := flgPreview || flgPreviewOnDemand || flgRedownload || flgRedownloadOne != "" || flgRebuildHTML || flgDeployDev || flgDeployProd || flgCiBuild || flgCiDaily
	if !hasCmd {
		flag.Usage()
		return
	}

	if flgRebuildHTML || flgCiBuild {
		// don't download from Notion, only read from cache
		d := getNotionCachingClient(false)
		d.RedownloadNewerVersions = false
		rebuildAll(d)
		return
	}

	if flgCiDaily {
		var cmd *exec.Cmd

		{
			// not sure if needed
			cmd = exec.Command("git", "checkout", "master")
			u.RunCmdLoggedMust(cmd)
		}

		// once a day re-download everything from Notion from scratch
		// checkin if files changed
		// and deploy to cloudflare if changed
		ghToken := os.Getenv("GITHUB_TOKEN")
		panicIf(ghToken == "", "GITHUB_TOKEN env variable missing")
		panicIf(os.Getenv("NOTION_TOKEN") == "", "NOTION_TOKEN env variable missing")
		panicIf(os.Getenv("CF_ACCOUNT_ID") == "", "CF_ACCOUNT_ID env variable missing")
		panicIf(os.Getenv("CF_API_TOKEN") == "", "CF_API_TOKEN env variable missing")
		d := getNotionCachingClient(true)
		rebuildAll(d)
		{
			cmd = exec.Command("git", "status")
			s := u.RunCmdMust(cmd)
			if strings.Contains(s, "nothing to commit, working tree clean") {
				// nothing changed so nothing else to do
				logf("Nothing changed, skipping deploy")
				return
			}
		}
		{
			// not sure if this is needed on GitHub CI
			cmd = exec.Command("git", "config", "--global", "user.email", "kkowalczyk@gmail.com")
			u.RunCmdLoggedMust(cmd)
			cmd = exec.Command("git", "config", "--global", "user.name", "Krzysztof Kowalczyk")
			u.RunCmdLoggedMust(cmd)
			/*
				cmd = exec.Command("git", "config", "--global", "github.user", "kjk")
				u.RunCmdLoggedMust(cmd)
				cmd = exec.Command("git", "config", "--global", "github.token", ghToken)
				u.RunCmdLoggedMust(cmd)
			*/

			cmd = exec.Command("git", "add", "notion_cache")
			u.RunCmdLoggedMust(cmd)
			nowStr := time.Now().Format("2006-01-02")
			commitMsg := "ci: update from onotion on " + nowStr
			cmd = exec.Command("git", "commit", "-am", commitMsg)
			u.RunCmdLoggedMust(cmd)

			if false {
				// TODO: do I need to be so specific or can I just do "git push"?
				s := strings.Replace("https://${GITHUB_TOKEN}@github.com/kjk/blog.git", "${GITHUB_TOKEN}", ghToken, -1)
				cmd = exec.Command("git", "push", s, "master")
				u.RunCmdLoggedMust(cmd)
			} else {
				cmd = exec.Command("git", "push")
				u.RunCmdLoggedMust(cmd)
			}
		}

		{
			cmd = exec.Command("wrangler", "publish")
			u.RunCmdLoggedMust(cmd)
		}
	}

	if flgRedownload {
		d := getNotionCachingClient(flgNoCache)
		rebuildAll(d)
		return
	}

	if flgRedownloadOne != "" {
		d := getNotionCachingClient(flgNoCache)
		_, err := d.DownloadPage(flgRedownloadOne)
		must(err)
		return
	}

	if flgDeployDev {
		d := getNotionCachingClient(flgNoCache)
		rebuildAll(d)
		cmd := exec.Command("wrangler", "publish")
		u.RunCmdLoggedMust(cmd)
		u.OpenBrowser("https://blog.kjk.workers.dev/")
		return
	}

	if flgDeployProd {
		d := getNotionCachingClient(flgNoCache)
		rebuildAll(d)
		cmd := exec.Command("wrangler", "publish", "-e", "production")
		u.RunCmdLoggedMust(cmd)
		u.OpenBrowser("https://blog.kowalczyk.info")
		return
	}

	if false {
		d := getNotionCachingClient(flgNoCache)
		testNotionToHTMLOnePage(d, "dfbefe6906a943d8b554699341e997b0")
		os.Exit(0)
	}

	d := getNotionCachingClient(flgNoCache)
	articles := rebuildAll(d)

	if flgPreview {
		preview()
		return
	}

	if flgPreviewOnDemand {
		startPreviewOnDemand(articles)
		return
	}

	flag.Usage()
}

func getNotionCachingClient(redownload bool) *notionapi.CachingClient {
	client := newNotionClient()
	d, err := notionapi.NewCachingClient(cacheDir, client)
	must(err)
	d.EventObserver = eventObserver
	d.RedownloadNewerVersions = true
	d.NoReadCache = redownload
	return d
}
