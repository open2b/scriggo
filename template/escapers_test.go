// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

import (
	"strings"
	"testing"
)

var urlEscapeCases = []struct {
	src      string
	expected string
}{
	{
		src:      `<a href="">`,
		expected: `<a href="">`,
	},

	{
		src:      `<a href="abc">`,
		expected: `<a href="abc">`,
	},
	{
		src:      `<a href="本">`,
		expected: `<a href="本">`,
	},
	{
		src:      "<a href=\"{{ `b` }}\">",
		expected: `<a href="b">`,
	},
	{
		src:      "<a href=\"{{ `/` }}\">",
		expected: `<a href="/">`,
	},
	{
		src:      "<a href=\"{{ `http://www.example.com/` }}\">",
		expected: `<a href="http://www.example.com/">`,
	},
	{
		src:      "<a href=\"{{ ` http://www.example.com/ ` }}\">",
		expected: `<a href=" http://www.example.com/ ">`,
	},
	{
		src:      "<a href=\"http://s/{{ `aà本` }}/\">",
		expected: `<a href="http://s/a%c3%a0%e6%9c%ac/">`,
	},
	{
		src:      "<a href=\"{{ `a` }}{{ `本` }}\">",
		expected: `<a href="a%e6%9c%ac">`,
	},
	{
		src:      "<a href=\"{{ `a` }}?b={{ ` ` }}\">",
		expected: `<a href="a?b=%20">`,
	},
	{
		src:      "<a href=\"{{ `a` }}?b={{ `=` }}\">",
		expected: `<a href="a?b=%3d">`,
	},
	{
		src:      "<a href=\"{{ `=` }}?b={{ `=` }}\">",
		expected: `<a href="=?b=%3d">`,
	},
	{
		src:      "<a href=\"{{ `p?` }}?b={{ `=` }}\">",
		expected: `<a href="p?b=%3d">`,
	},
	{
		src:      "<a href=\"{{ `p?q` }}?b={{ `=` }}\">",
		expected: `<a href="p?q&amp;b=%3d">`,
	},
	{
		src:      "<a href=\"{{ `?` }}?b={{ `=` }}\">",
		expected: `<a href="?b=%3d">`,
	},
	{
		src:      "<a href=\"{{ `/a/b/c` }}?b={{ `=` }}&c={{ `6` }}\">",
		expected: `<a href="/a/b/c?b=%3d&c=6">`,
	},
	{
		src:      "<a href=\"{{ `/a/b/c` }}?b={{ `=` }}&amp;c={{ `6` }}\">",
		expected: `<a href="/a/b/c?b=%3d&amp;c=6">`,
	},
	{
		src:      "<a href=\"{{ `` }}?{{ `` }}\">",
		expected: `<a href="?">`,
	},
	{
		src:      "<a href=\"?{{ `b` }}\">",
		expected: `<a href="?b">`,
	},
	{
		src:      "<a href=\"?{{ `=` }}\">",
		expected: `<a href="?%3d">`,
	},
	{
		src:      `<a href="#">`,
		expected: `<a href="#">`,
	},
	{
		src:      "<a href=\"#{{ `=` }}\">",
		expected: `<a href="#%3d">`,
	},
	{
		src:      "<a href=\"{{ `=` }}#{{ `=` }}\">",
		expected: `<a href="=#%3d">`,
	},
	{
		src:      "<a href=\"{{ `=` }}?{{ `=` }}#{{ `=` }}\">",
		expected: `<a href="=?%3d#%3d">`,
	},
	{
		src:      "<a href=\"{{ `=` }}?b=6#{{ `=` }}\">",
		expected: `<a href="=?b=6#%3d">`,
	},
	{
		src:      "<a href=\"{{ `,` }}?{{ `,` }}\">",
		expected: `<a href=",?%2c">`,
	},
	{
		src:      "<img srcset=\"{{ `large.jpg` }} 1024w, {{ `medium.jpg` }} 640w,{{ `small.jpg` }} 320w\">",
		expected: `<img srcset="large.jpg 1024w, medium.jpg 640w,small.jpg 320w">`,
	},
	{
		src:      "<img srcset=\"{{ `large.jpg?s=1024` }} 1024w, {{ `medium.jpg` }} 640w\">",
		expected: `<img srcset="large.jpg?s=1024 1024w, medium.jpg 640w">`,
	},
	{
		src:      "<img srcset=\"{{ `large.jpg?s=1024` }} 1024w, {{ `medium=.jpg` }} 640w\">",
		expected: `<img srcset="large.jpg?s=1024 1024w, medium=.jpg 640w">`,
	},
	{
		src:      "<a href=\"{% if true %}{{ `=` }}{% else %}?{{ `=` }}{% end %}\">",
		expected: `<a href="=">`,
	},
	{
		src:      "<a href=\"{% if false %}{{ `=` }}{% else %}?{{ `=` }}{% end %}\">",
		expected: `<a href="?%3d">`,
	},
	{
		src:      "<input {{ `disabled` }}>",
		expected: `<input disabled>`,
	},
	{
		src:      "<a href={{ `b` }}>",
		expected: `<a href=b>`,
	},
	{
		src:      "<a href={{ ` b ` }}>",
		expected: `<a href=&#32;b&#32;>`,
	},
	{
		src:      "<a href= {{ ` b `}} >",
		expected: `<a href= &#32;b&#32; >`,
	},
	{
		src:      "<a href= {{ \"\\t\\n\\r\\x0C b=`\" }} >",
		expected: `<a href= %09%0a%0d%0c&#32;b=%60 >`,
	},
	{
		src:      "<a href=\"{{ (map[interface{}]interface{}{})[`a`] }}\">",
		expected: `<a href="">`,
	},
	// TODO: enable this test:
	// {
	// 	src:      "<a href=\"{{ `p?&` }}?b={{ `=` }}\">",
	// 	expected: `<a href="p?&amp;b=%3d">`,
	// },
}

func TestURLEscape(t *testing.T) {
	for _, cas := range urlEscapeCases {
		t.Run("", func(t *testing.T) {
			r := MapReader{"/index.html": []byte(cas.src)}
			templ, err := Load("/index.html", r, Builtins(), ContextHTML, nil)
			if err != nil {
				t.Fatalf("compilation error: %s", err)
			}
			out := &strings.Builder{}
			err = templ.Render(out, nil, nil)
			if err != nil {
				t.Fatalf("rendering error: %s", err)
			}
			got := out.String()
			if got != cas.expected {
				t.Fatalf("src: %q: expecting %q, got %q", cas.src, cas.expected, got)
			}
		})

	}
}
