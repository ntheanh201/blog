package main

import (
	"flag"
	"os/exec"
)

var (
	flgBuild bool
	flgWc    bool
)

func parseFlags() {
	flag.BoolVar(&flgBuild, "build", false, "build the executable")
	flag.BoolVar(&flgWc, "wc", false, "wc -l i.e. line count")
	flag.Parse()
}

func build() {
	cmd := exec.Command("go", "build", ".")
	runCmd(cmd)
}

func main() {
	cdUpDir("blog")
	logf("currDir: '%s'\n", currDirAbs())

	parseFlags()

	if flgBuild {
		build()
		return
	}

	if flgWc {
		doLineCount()
		return
	}

	flag.Usage()
}
