package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/kjk/u"
)

func notDataDir(name string) bool {
	name = filepath.ToSlash(name)
	return !strings.Contains(name, "tmpdata/")
}

var srcFiles = u.MakeAllowedFileFilterForExts(".go")
var allFiles = u.MakeFilterAnd(srcFiles, notDataDir)

func doLineCount() int {
	stats := u.NewLineStats()
	err := stats.CalcInDir(".", srcFiles, true)
	if err != nil {
		fmt.Printf("doWordCount: stats.wcInDir() failed with '%s'\n", err)
		return 1
	}
	u.PrintLineStats(stats)
	return 0
}
