package main

import (
	"testing"

	"github.com/kjk/common/assert"
)

func TestTrimEmptyLines(t *testing.T) {
	tests := []struct {
		a   []string
		exp []string
	}{
		{
			a:   []string{"a"},
			exp: []string{"a"},
		},
		{
			a:   []string{"a", "", "", "b"},
			exp: []string{"a", "", "b"},
		},
		{
			a:   []string{"", "a", ""},
			exp: []string{"a"},
		},
		{
			a:   []string{"", "", "a", "", "b", "", ""},
			exp: []string{"a", "", "b"},
		},
	}
	for _, test := range tests {
		got := trimEmptyLines(test.a)
		assert.Equal(t, test.exp, got)
	}
}

func TestRemoveHashtags(t *testing.T) {
	tests := []struct {
		s    string
		tags []string
		sExp string
	}{
		{
			s:    "#idea Build a web service  ",
			sExp: "Build a web service",
			tags: []string{"idea"},
		},
		{
			s:    "#foo   #BAr and #me",
			sExp: "and",
			tags: []string{"foo", "bar", "me"},
		},
		{
			s:    "not #found here",
			sExp: "not #found here",
			tags: nil,
		},
		{
			s:    "#foo   not a#hash",
			sExp: "not a#hash",
			tags: []string{"foo"},
		},
	}
	for _, test := range tests {
		sGot, tags := removeHashTags(test.s)
		assert.Equal(t, test.sExp, sGot)
		assert.Equal(t, test.tags, tags)
	}
}
