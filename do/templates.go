package main

import (
	"bytes"
	"html/template"
	"io"
	"io/ioutil"
	"path/filepath"
)

var (
	templates *template.Template
)

func loadTemplates() {
	pattern := filepath.Join("www", "tmpl", "*.tmpl.html")
	templates = template.Must(template.ParseGlob(pattern))
}

func execTemplateToFile(path string, templateName string, model interface{}) error {
	var buf bytes.Buffer
	err := templates.ExecuteTemplate(&buf, templateName, model)
	must(err)
	err = ioutil.WriteFile(path, buf.Bytes(), 0644)
	return err
}

func execTemplateToWriter(name string, data interface{}, w io.Writer) error {
	loadTemplates() // TODO: only reload when changed
	return templates.ExecuteTemplate(w, name, data)
}

func execTemplate(path string, tmplName string, d interface{}, w io.Writer) error {
	// this code path is for the preview on demand server
	if w != nil {
		return execTemplateToWriter(tmplName, d, w)
	}

	// this code path is for generating static files
	netPath := wwwPath(path)
	err := execTemplateToFile(netPath, tmplName, d)
	must(err)
	return nil
}
