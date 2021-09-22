package main

import (
	"context"
	"fmt"
	"os"
)

var (
	logFile *os.File
)

func openLog() {
	var err error
	logFile, err = os.Create("log.txt")
	must(err)
}

func closeLog() {
	if logFile == nil {
		return
	}
	logFile.Close()
	logFile = nil
}

/*
// TODO: should take additional format and args for optional message
func logError(err error) {
	if err != nil {
		return
	}
	logf(ctx(), "%s", err.Error())
}
*/

func logf(ctx context.Context, format string, args ...interface{}) {
	s := fmt.Sprintf(format, args...)
	if logFile != nil {
		fmt.Fprint(logFile, s)
	}
	fmt.Print(s)
}

func logvf(format string, args ...interface{}) {
	s := fmt.Sprintf(format, args...)
	if logFile != nil {
		fmt.Fprint(logFile, s)
	}
}

var (
	doTempLog = false
)

func logTemp(format string, args ...interface{}) {
	if !doTempLog {
		return
	}
	logf(ctx(), format, args...)
}
