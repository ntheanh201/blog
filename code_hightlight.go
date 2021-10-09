package main

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/kjk/common/httputil"
)

var validThemes = []string{"abap", "algol", "algol_nu", "arduino", "autumn", "borland", "bw", "colorful", "dracula", "emacs", "friendly", "fruity", "github", "igor", "lovelace", "manni", "monokai", "monokailight", "murphy", "native", "paraiso-dark", "paraiso-light", "pastie", "perldoc", "pygments", "rainbow_dash", "rrt", "solarized-dark", "solarized-dark256", "solarized-light", "swapoff", "tango", "trac", "vim", "vs", "xcode"}

var defaultTheme = "monokailight"

func validateTheme(theme string) string {
	if theme == "" {
		return defaultTheme
	}
	theme = strings.ToLower(theme)
	for _, t := range validThemes {
		if t == theme {
			return theme
		}
	}
	return defaultTheme
}

func makeHTMLFormatter(noLines bool) chroma.Formatter {
	opts := []html.Option{
		//html.Standalone(),
		html.TabWidth(4),
	}
	if !noLines {
		opts = append(opts, html.WithLineNumbers(true))
		opts = append(opts, html.LineNumbersInTable(true))
	}
	return html.New(opts...)
}

func codeHighlight(w io.Writer, source string, fileName string, formatter chroma.Formatter, style string) error {
	style = validateTheme(style)
	l := lexers.Match(fileName)
	if l == nil {
		l = lexers.Analyse(source)
	}
	if l == nil {
		l = lexers.Fallback
	}
	l = chroma.Coalesce(l)

	s := styles.Get(style)
	if s == nil {
		s = styles.Fallback
	}

	it, err := l.Tokenise(nil, source)
	if err != nil {
		return err
	}
	return formatter.Format(w, s, it)
}

func testCodeHighlight() {
	rawURL := getGitHubRawURL("https://github.com/essentialbooks/books/blob/master/books/go/0010-getting-started/hello_world.go")
	fmt.Printf("rawURL: %s\n", rawURL)
	timeStart := time.Now()
	d, err := httputil.Get(rawURL)
	dur := time.Since(timeStart)
	if err != nil {
		logerrf(ctx(), "Failed to download %s. Time: %s. Error: %s\n", rawURL, dur, err)
		return
	}
	var buf bytes.Buffer
	f := makeHTMLFormatter(false)
	err = codeHighlight(&buf, string(d), "hello_world.go", f, "monokailight")
	if err != nil {
		logerrf(ctx(), "quick.Highlight failed with %s\n", err)
		return
	}
	fmt.Printf("html:\n%s\n", buf.Bytes())
}
