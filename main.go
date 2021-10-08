package main

import (
	"flag"
	"log"
	_ "net/url"
	"os"
	"os/exec"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/kjk/notionapi"
)

var (
	dirWwwGenerated = "www_generated" // directory where we generate html files
	httpPort        = 9044

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

var (
	cachingPolicy = notionapi.PolicyDownloadNewer
)

func main() {
	var (
		flgRun             bool
		flgRunProd         bool
		flgImportNotion    bool
		flgGen             bool
		flgDiff            bool
		flgCiDaily         bool
		flgImportNotionOne string
		flgProfile         string
	)

	{
		// flag.BoolVar(&flgWc, "wc", false, "wc -l i.e. line count")
		flag.BoolVar(&flgVerbose, "verbose", false, "if true, verbose logging")
		flag.BoolVar(&flgNoCache, "no-cache", false, "if true, disables cache for downloading notion pages")
		flag.BoolVar(&flgRun, "run", false, "run server locally")
		flag.BoolVar(&flgRunProd, "run-prod", false, "run server in production")
		flag.BoolVar(&flgImportNotion, "import-notion", false, "re-download the content from Notion. use -no-cache to disable cache")
		flag.BoolVar(&flgGen, "gen", false, "gen html in www_generated/ directory")
		//flag.BoolVar(&flgDiff, "diff", false, "preview diff using winmerge")
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
		optimizeAllImages([]string{"notion_cache", "www"})
		return
	}

	if false {
		flgImportNotionOne = "08e19004306b413aba6e0e86a10fec7a"
	}

	// for those commands we only want to use cache
	if flgGen || flgRun {
		cachingPolicy = notionapi.PolicyCacheOnly
	}

	if flgRun {
		runServer()
		return
	}

	if flgRunProd {
		runServerProd()
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
		cachingPolicy = notionapi.PolicyCacheOnly
		genHTMLServer(dirWwwGenerated)
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

	if flgGen {
		cachingPolicy = notionapi.PolicyCacheOnly
		genHTMLServer(dirWwwGenerated)
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
