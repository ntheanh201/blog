package main

import (
	"archive/zip"
	"bytes"
	"compress/flate"
	"io"
)

// upload to instantpreview.dev

func zipWriteContent(zw *zip.Writer, files map[string][]byte) error {
	for name, data := range files {
		fw, err := zw.Create(name)
		if err != nil {
			return err
		}
		_, err = fw.Write(data)
		if err != nil {
			return err
		}
	}
	return zw.Close()
}

func zipCreateFromContent(files map[string][]byte) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	zw.RegisterCompressor(zip.Deflate, func(out io.Writer) (io.WriteCloser, error) {
		return flate.NewWriter(out, flate.BestCompression)
	})
	err := zipWriteContent(zw, files)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func uploadFilesToInstantPreviewMust(files map[string][]byte) string {
	uri := "https://instantpreview.dev/upload"
	zipData, err := zipCreateFromContent(files)
	must(err)
	res, err := httpPost(uri, zipData)
	must(err)
	return string(res)
}
