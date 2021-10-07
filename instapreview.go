package main

import (
	"time"
)

// upload to https://www.instantpreview.dev

func uploadFilesToInstantPreviewMust(files map[string][]byte) string {
	zipData, err := zipCreateFromContent(files)
	must(err)
	return uploadZipToInstantPreviewMust(zipData)
}

func uploadZipToInstantPreviewMust(zipData []byte) string {
	timeStart := time.Now()
	uri := "https://www.instantpreview.dev/upload"
	res, err := httpPost(uri, zipData)
	must(err)
	uri = string(res)
	sizeStr := formatSize(int64(len(zipData)))
	logf(ctx(), "uploaded under: %s, zip file size: %s in: %s\n", uri, sizeStr, time.Since(timeStart))
	return uri
}

func uploadServerToInstantPreviewMust(handlers []Handler) string {
	zipData, err := WriteServerFilesToZip(handlers)
	must(err)
	return uploadZipToInstantPreviewMust(zipData)
}
