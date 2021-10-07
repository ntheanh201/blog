package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	_ "net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/kjk/notionapi"
)

var (
	generatedHTMLDir = "www_generated" // directory where we generate html files

	flgVerbose bool
	flgNoCache bool
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

func runWranglerDev() {
	err := exec.Command("wrangler", "--version").Run()
	if err != nil {
		err = exec.Command("npm", "i", "-g", "@cloudflare/wrangler").Run()
		panicIfErr(err)
	}
	cmd := exec.Command("wrangler", "dev")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	logIfError(err)
}

func hasWranglerConfig() bool {
	homeDir, err := os.UserHomeDir()
	panicIfErr(err)
	wranglerConfigPath := filepath.Join(homeDir, ".wrangler", "config", "default.toml")
	if _, err := os.Stat(wranglerConfigPath); err == nil {
		logf(ctx(), "hasWranglerConfig: yes\n")
		return true
	}
	apiKey := strings.TrimSpace(os.Getenv("CLOUDFLARE_API_TOKEN"))
	if apiKey == "" {
		return false
	}
	must(createDirForFile(wranglerConfigPath))
	toml := fmt.Sprintf(`api_token = "%s"`+"\n", apiKey)
	must(ioutil.WriteFile(wranglerConfigPath, []byte(toml), 0644))
	return true
}

var (
	cachingPolicy = notionapi.PolicyDownloadNewer
)

func main() {
	var (
		flgPreviewWrangler  bool
		flgPreviewServer    bool
		flgPreviewInstaPrev bool
		flgDeployDev        bool
		flgDeployProd       bool
		flgImportNotion     bool
		flgRebuildHTML      bool
		flgDiff             bool
		flgCiDaily          bool
		flgCiBuild          bool
		flgImportNotionOne  string
		flgProfile          string
	)

	{
		// flag.BoolVar(&flgWc, "wc", false, "wc -l i.e. line count")
		flag.BoolVar(&flgVerbose, "verbose", false, "if true, verbose logging")
		flag.BoolVar(&flgNoCache, "no-cache", false, "if true, disables cache for downloading notion pages")
		flag.BoolVar(&flgDeployDev, "deploy-dev", false, "deploy to https://blog.kjk.workers.dev/")
		flag.BoolVar(&flgDeployProd, "deploy-prod", false, "deploy to https://blog.kowalczyk.info")
		flag.BoolVar(&flgPreviewServer, "preview-server", false, "preview with web server running locally")
		flag.BoolVar(&flgPreviewWrangler, "preview-wrangler", false, "preview with wrangler")
		flag.BoolVar(&flgPreviewInstaPrev, "preview-insta", false, "preview with instant preview")
		flag.BoolVar(&flgImportNotion, "import-notion", false, "re-download the content from Notion. use -no-cache to disable cache")
		flag.BoolVar(&flgRebuildHTML, "rebuild", false, "rebuild html in www_generated/ directory")
		//flag.BoolVar(&flgDiff, "diff", false, "preview diff using winmerge")
		flag.BoolVar(&flgCiBuild, "ci-build", false, "runs on GitHub CI for every checkin")
		flag.BoolVar(&flgCiDaily, "ci-daily", false, "runs once a day on GitHub CI")
		//flag.StringVar(&flgProfile, "profile", "", "name of file to save cpu profiling info")
		flag.Parse()
	}

	timeStart := time.Now()
	defer func() {
		logf(ctx(), "finished in %s\n", time.Since(timeStart))
	}()

	cdUpDir("blog")

	if false {
		dirToLF(".")
		return
	}

	if false {
		flgImportNotionOne = "08e19004306b413aba6e0e86a10fec7a"
	}

	// for those commands we only want to use cache
	if flgPreviewWrangler || flgRebuildHTML || flgCiBuild || flgPreviewServer {
		cachingPolicy = notionapi.PolicyCacheOnly
	}

	if false {
		flgPreviewWrangler = true
		cachingPolicy = notionapi.PolicyDownloadNewer
	}

	openLog()
	defer closeLog()

	if flgPreviewServer {
		runServer()
		return
	}

	if flgDiff {
		winmergeDiffPreview()
		return
	}

	if flgProfile != "" {
		logf(ctx(), "staring cpu profile in file '%s'\n", flgProfile)
		f, err := os.Create(flgProfile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	if flgCiDaily {
		var cmd *exec.Cmd

		{
			// not sure if needed
			cmd = exec.Command("git", "checkout", "master")
			runCmdLoggedMust(cmd)
		}

		// once a day re-download everything from Notion from scratch
		// checkin if files changed
		// and deploy to cloudflare if changed
		ghToken := os.Getenv("GITHUB_TOKEN")
		panicIf(ghToken == "", "GITHUB_TOKEN env variable missing")
		panicIf(os.Getenv("NOTION_TOKEN") == "", "NOTION_TOKEN env variable missing")
		panicIf(os.Getenv("CF_ACCOUNT_ID") == "", "CF_ACCOUNT_ID env variable missing")
		panicIf(os.Getenv("CF_API_TOKEN") == "", "CF_API_TOKEN env variable missing")
		cachingPolicy = notionapi.PolicyCacheOnly
		genHTMLServer(generatedHTMLDir)
		{
			cmd = exec.Command("git", "status")
			s := runCmdMust(cmd)
			if strings.Contains(s, "nothing to commit, working tree clean") {
				// nothing changed so nothing else to do
				logf(ctx(), "Nothing changed, skipping deploy")
				return
			}
		}
		{
			// not sure if this is needed on GitHub CI
			cmd = exec.Command("git", "config", "--global", "user.email", "kkowalczyk@gmail.com")
			runCmdLoggedMust(cmd)
			cmd = exec.Command("git", "config", "--global", "user.name", "Krzysztof Kowalczyk")
			runCmdLoggedMust(cmd)
			/*
				cmd = exec.Command("git", "config", "--global", "github.user", "kjk")
				runCmdLoggedMust(cmd)
				cmd = exec.Command("git", "config", "--global", "github.token", ghToken)
				runCmdLoggedMust(cmd)
			*/

			cmd = exec.Command("git", "add", "notion_cache")
			runCmdLoggedMust(cmd)
			nowStr := time.Now().Format("2006-01-02")
			commitMsg := "ci: update from notion on " + nowStr
			cmd = exec.Command("git", "commit", "-am", commitMsg)
			runCmdLoggedMust(cmd)

			if false {
				// TODO: do I need to be so specific or can I just do "git push"?
				s := strings.Replace("https://${GITHUB_TOKEN}@github.com/kjk/blog.git", "${GITHUB_TOKEN}", ghToken, -1)
				cmd = exec.Command("git", "push", s, "master")
				runCmdLoggedMust(cmd)
			} else {
				cmd = exec.Command("git", "push")
				runCmdLoggedMust(cmd)
			}
		}

		{
			cmd = exec.Command("wrangler", "publish")
			runCmdLoggedMust(cmd)
		}
		return
	}

	if flgImportNotion {
		cachingPolicy = notionapi.PolicyDownloadNewer
		cc := getNotionCachingClient()
		_ = loadArticles(cc)
		return
	}

	if flgImportNotionOne != "" {
		cachingPolicy = notionapi.PolicyDownloadAlways
		cc := getNotionCachingClient()
		_, err := cc.DownloadPage(flgImportNotionOne)
		must(err)
		return
	}

	if flgDeployDev {
		panicIf(!hasWranglerConfig(), "no wrangler config!")
		cachingPolicy = notionapi.PolicyCacheOnly
		genHTMLServer(generatedHTMLDir)
		cmd := exec.Command("wrangler", "publish")
		runCmdLoggedMust(cmd)
		openBrowser("https://blog.kjk.workers.dev/")
		return
	}

	if flgDeployProd {
		panicIf(!hasWranglerConfig(), "no wrangler config!")
		cachingPolicy = notionapi.PolicyCacheOnly
		genHTMLServer(generatedHTMLDir)
		cmd := exec.Command("wrangler", "publish", "-e", "production")
		runCmdLoggedMust(cmd)
		openBrowser("https://blog.kowalczyk.info")
		return
	}

	if flgCiBuild {
		cachingPolicy = notionapi.PolicyCacheOnly
		genHTMLServer(generatedHTMLDir)
		return
	}

	if flgRebuildHTML {
		cachingPolicy = notionapi.PolicyCacheOnly
		genHTMLServer(generatedHTMLDir)
		return
	}

	if flgPreviewInstaPrev {
		server := makeDynamicServer()
		uri := uploadServerToInstantPreviewMust(server.Handlers)
		logf(ctx(), "uploaded to '%s'\n", uri)
		openBrowser(uri)
		return
	}

	if flgPreviewWrangler {
		genHTMLServer(generatedHTMLDir)
		if isWindows() || hasWranglerConfig() {
			runWranglerDev()
			return
		}
		runServer()
		return
	}

	flag.Usage()
}

func getNotionCachingClient() *notionapi.CachingClient {
	if flgNoCache {
		cachingPolicy = notionapi.PolicyDownloadAlways
	}
	token := os.Getenv("NOTION_TOKEN")
	if token == "" && cachingPolicy != notionapi.PolicyCacheOnly {
		logf(ctx(), "must set NOTION_TOKEN env variable\n")
		os.Exit(1)
	}
	// TODO: verify token still valid, somehow
	client := &notionapi.Client{
		AuthToken: token,
	}
	if flgVerbose {
		client.Logger = os.Stdout
		client.DebugLog = true
	}

	d, err := notionapi.NewCachingClient(cacheDir, client)
	must(err)
	d.Policy = cachingPolicy
	return d
}
