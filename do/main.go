package main

import (
	"flag"
	"os"
	"os/exec"

	"github.com/kjk/u"
)

var (
	flgBuild         bool
	flgPreview       bool
	flgPreviewStatic bool
	flgWc            bool
)

func parseFlags() {
	flag.BoolVar(&flgBuild, "build", false, "build the executable")
	flag.BoolVar(&flgWc, "wc", false, "wc -l i.e. line count")
	flag.BoolVar(&flgPreview, "preview", false, "preview on demand (rebuild html on access)")
	flag.BoolVar(&flgPreviewStatic, "preview-static", false, "preview static (rebuild html first)")
	flag.Parse()
}

func build() {
	cmd := exec.Command("go", "build", "-o", "blog_app.exe", ".")
	u.RunCmdMust(cmd)
}

func preview() {
	exeName := "./blog_app.exe"
	cmd := exec.Command("go", "build", "-o", exeName, ".")
	u.RunCmdMust(cmd)
	cmd = exec.Command(exeName, "-preview-on-demand")
	defer os.Remove(exeName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	u.RunCmdMust(cmd)
}

func previewStatic() {
	exeName := "./blog_app.exe"
	cmd := exec.Command("go", "build", "-o", exeName, ".")
	u.RunCmdMust(cmd)
	cmd = exec.Command(exeName, "-preview")
	defer os.Remove(exeName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	u.RunCmdMust(cmd)
}

func main() {
	u.CdUpDir("blog")
	u.Logf("currDir: '%s'\n", u.CurrDirAbsMust())

	parseFlags()

	if flgBuild {
		build()
		return
	}

	if flgPreview {
		preview()
		return
	}

	if flgPreviewStatic {
		previewStatic()
		return
	}

	if flgWc {
		doLineCount()
		return
	}

	flag.Usage()
}
