package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGitHubRawURL(t *testing.T) {
	tests := []struct {
		s   string
		exp string
	}{
		{
			"github.com/essentialbooks/books/blob/master/netlify.toml", "https://raw.githubusercontent.com/essentialbooks/books/master/netlify.toml",
		},
		{
			"github2.com/essentialbooks/books/blob/master/netlify.toml",
			"",
		},
		{
			"github.com/essentialbooks/books/blob/master/books/go/0010-getting-started/hello_world.go",
			"https://raw.githubusercontent.com/essentialbooks/books/master/books/go/0010-getting-started/hello_world.go",
		},
		{
			"github.com/essentialbooks/books/blob/371e8cbbef7641649d875c7089033e9460cf4fee/books/go/0010-getting-started/010-install.md",
			"https://raw.githubusercontent.com/essentialbooks/books/371e8cbbef7641649d875c7089033e9460cf4fee/books/go/0010-getting-started/010-install.md",
		},
	}
	for _, test := range tests {
		got := getGitHubRawURL(test.s)
		assert.Equal(t, test.exp, got)
	}
}
