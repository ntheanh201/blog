package main

import (
	"context"
	"fmt"
)

var (
	logVerbose = false
)

func logf(ctx context.Context, s string, args ...interface{}) {
	if len(args) > 0 {
		s = fmt.Sprintf(s, args...)
	}
	fmt.Print(s)
}

func logerrf(ctx context.Context, format string, args ...interface{}) {
	s := format
	if len(args) > 0 {
		s = fmt.Sprintf(format, args...)
	}
	fmt.Printf("Error: %s", s)
}

func logvf(s string, args ...interface{}) {
	if !logVerbose {
		return
	}

	if len(args) > 0 {
		s = fmt.Sprintf(s, args...)
	}
	fmt.Print(s)
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
