package main

import (
	"flag"
	"os"
	"os/exec"

	"github.com/kjk/u"
)

var (
	exeName = "./blog_app.exe"
)

func build() {
	cmd := exec.Command("go", "build", "-o", exeName, ".")
	u.RunCmdMust(cmd)
}

func runWithArgs(args ...string) {
	build()
	cmd := exec.Command(exeName, args...)
	defer os.Remove(exeName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	u.RunCmdMust(cmd)
}

func main() {
	u.CdUpDir("blog")
	u.Logf("currDir: '%s'\n", u.CurrDirAbsMust())

	var (
		flgBuild         bool
		flgPreview       bool
		flgPreviewStatic bool
		flgDeployDraft   bool
		flgDeployProd    bool
		flgRedownload    bool
		flgWc            bool
	)

	flag.BoolVar(&flgBuild, "build", false, "build the executable")
	flag.BoolVar(&flgWc, "wc", false, "wc -l i.e. line count")
	flag.BoolVar(&flgPreview, "preview", false, "preview on demand (rebuild html on access)")
	flag.BoolVar(&flgPreviewStatic, "preview-static", false, "preview static (rebuild html first)")
	flag.BoolVar(&flgDeployDraft, "deploy-draft", false, "deploy to netlify as draft")
	flag.BoolVar(&flgDeployProd, "deploy-prod", false, "deploy to netlify production")
	flag.BoolVar(&flgRedownload, "redownload", false, "redownload from notion")
	flag.Parse()

	if flgBuild {
		build()
		return
	}

	if flgPreview {
		runWithArgs("-preview-on-demand")
		return
	}

	if flgPreviewStatic {
		runWithArgs("-preview")
		return
	}

	if flgRedownload {
		runWithArgs("-redownload-notion")
	}

	if flgDeployDraft {
		runWithArgs("-preview")
		cmd := exec.Command("netlify", "deploy", "--dir=netlify_static", "--site=a1bb4018-531d-4de8-934d-8d5602bacbfb", "--open")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		u.RunCmdMust(cmd)
		return
	}

	if flgDeployProd {
		runWithArgs("-preview")
		cmd := exec.Command("netlify", "deploy", "--prod", "--dir=netlify_static", "--site=a1bb4018-531d-4de8-934d-8d5602bacbfb", "--open")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		u.RunCmdMust(cmd)
		return
	}

	if flgWc {
		doLineCount()
		return
	}

	flag.Usage()
}
