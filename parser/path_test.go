// Copyright (c) 2018 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parser

import (
	"testing"
)

const long255chars = "È1234567890123456789012345678901234567890123456789012" +
	"È123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789" +
	"È123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789.a"

const long256chars = "È12345678901234567890123456789012345678901234567890123" +
	"È123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789" +
	"È123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789.a"

var validFilePaths = []string{"/a.b", "/a/a.b", "a.b", "a/a.b", "../a.b", "a/../a.b", long255chars}

var invalidFilePaths = []string{"", "/", "a/", ".", "..", "...", ".aa", "a/a..", "a/ab.", "./abc",
	" ", " abc", "abc ", "a\x00b.c", "a\x1fb.c", "a\"b.c", "a*ab.c", "a:b.c", "a<b.c",
	"a>b.c", "a?b.c", "a\\b.c", "a|b.c", "a\x7fb.c", "/con/ab.c", "/prn/ab.c", "/aux/ab.c", "/nul/ab.c",
	"com0/ab.c", "com9/ab.c", "lpt0/ab.c", "lpt9/ab.c", "con.a", "prn.a", "aux.a", "nul.a", "com0.a", "com9.a",
	"lpt0.a", "lpt9.a", long256chars,
}

func TestValidFilePath(t *testing.T) {
	for _, p := range validFilePaths {
		if !isValidFilePath(p) {
			t.Errorf("path: %q, expected valid, but invalid\n", p)
		}
	}
	for _, p := range invalidFilePaths {
		if isValidFilePath(p) {
			t.Errorf("path: %q, expected invalid, but valid\n", p)
		}
	}
}
