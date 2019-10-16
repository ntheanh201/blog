package main

import (
	"flag"
	"fmt"
	"log"
	_ "net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/kjk/notionapi"
	"github.com/kjk/notionapi/caching_downloader"
	"github.com/kjk/u"
)

const (
	analyticsCode = "UA-194516-1"
)

var (
	flgVerbose bool
)

func rebuildAll(d *caching_downloader.Downloader) *Articles {
	regenMd()
	loadTemplates()
	articles := loadArticles(d)
	readRedirects(articles)
	netlifyBuild(articles)
	return articles
}

// caddy -log stdout
func runCaddy() {
	cmd := exec.Command("caddy", "-log", "stdout")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

/*
func stopCaddy(cmd *exec.Cmd) {
	cmd.Process.Kill()
}
*/

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

func preview() {
	go func() {
		time.Sleep(time.Second * 1)
		openBrowser("http://localhost:8080")
	}()
	runCaddy()
}

var (
	nDownloadedPage = 0
)

func eventObserver(ev interface{}) {
	switch v := ev.(type) {
	case *caching_downloader.EventError:
		logf(v.Error)
	case *caching_downloader.EventDidDownload:
		nDownloadedPage++
		logf("%03d '%s' : downloaded in %s\n", nDownloadedPage, v.PageID, v.Duration)
	case *caching_downloader.EventDidReadFromCache:
		// TODO: only verbose
		nDownloadedPage++
		logf("%03d '%s' : read from cache in %s\n", nDownloadedPage, v.PageID, v.Duration)
	case *caching_downloader.EventGotVersions:
		logf("downloaded info about %d versions in %s\n", v.Count, v.Duration)
	}
}

func newNotionClient() *notionapi.Client {
	token := os.Getenv("NOTION_TOKEN")
	if token == "" {
		logf("must set NOTION_TOKEN env variable\n")
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

func cmdAddNetlifyToken(cmd *exec.Cmd) {
	token := os.Getenv("NETLIFY_TOKEN")
	if token == "" {
		logf("No NETLIFY_TOKEN\n")
		return
	}
	logf("Has NETLIFY_TOKEN\n")
	cmd.Args = append(cmd.Args, "--auth", token)
}

func main() {
	var (
		flgDeployDraft     bool
		flgDeployProd      bool
		flgPreview         bool
		flgPreviewOnDemand bool
		flgNoCache         bool
		flgWc              bool
	)

	{
		flag.BoolVar(&flgWc, "wc", false, "wc -l i.e. line count")
		flag.BoolVar(&flgVerbose, "verbose", false, "if true, verbose logging")
		flag.BoolVar(&flgNoCache, "no-cache", false, "if true, disables cache for downloading notion pages")
		flag.BoolVar(&flgDeployDraft, "deploy-draft", false, "deploy to netlify as draft")
		flag.BoolVar(&flgDeployProd, "deploy-prod", false, "deploy to netlify production")
		flag.BoolVar(&flgPreview, "preview", false, "if true, runs caddy and opens a browser for preview")
		flag.BoolVar(&flgPreviewOnDemand, "preview-on-demand", false, "if true runs the browser for local preview")
		flag.Parse()
	}

	openLog()
	defer closeLog()

	recreateDir("netlify_static")

	if flgWc {
		doLineCount()
		return
	}

	client := newNotionClient()
	cache, err := caching_downloader.NewDirectoryCache(cacheDir)
	must(err)
	d := caching_downloader.New(cache, client)
	d.EventObserver = eventObserver
	d.RedownloadNewerVersions = true
	d.NoReadCache = flgNoCache

	doOpen := runtime.GOOS == "darwin"
	//os.Setenv("PATH", )
	netlifyExe := filepath.Join("./node_modules", ".bin", "netlify")
	if flgDeployDraft {
		rebuildAll(d)
		cmd := exec.Command(netlifyExe, "deploy", "--dir=netlify_static", "--site=a1bb4018-531d-4de8-934d-8d5602bacbfb")
		cmdAddNetlifyToken(cmd)
		if doOpen {
			cmd.Args = append(cmd.Args, "--open")
		}
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		u.RunCmdMust(cmd)
		return
	}

	if flgDeployProd {
		rebuildAll(d)
		cmd := exec.Command(netlifyExe, "deploy", "--prod", "--dir=netlify_static", "--site=a1bb4018-531d-4de8-934d-8d5602bacbfb")
		cmdAddNetlifyToken(cmd)
		if doOpen {
			cmd.Args = append(cmd.Args, "--open")
		}
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		u.RunCmdMust(cmd)
		return
	}

	if false {
		testNotionToHTMLOnePage(d, "dfbefe6906a943d8b554699341e997b0")
		os.Exit(0)
	}

	articles := rebuildAll(d)

	if flgPreview {
		preview()
		return
	}

	if flgPreviewOnDemand {
		startPreviewOnDemand(articles)
		return
	}
}
