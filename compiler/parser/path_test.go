// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parser

import (
	"testing"
)

var validPaths = []string{"a", "/a", "a/b", "/a/b", "../a", "a/../a", ".", "..."}

var invalidPaths = []string{"\xf0", "", "..", "/", "a/", "//", "a//", "//a",
	"/..", "a/.."}

func TestValidPath(t *testing.T) {
	for _, p := range validPaths {
		if !validPath(p) {
			t.Errorf("path: %q, expected valid, but invalid\n", p)
		}
	}
	for _, p := range invalidPaths {
		if validPath(p) {
			t.Errorf("path: %q, expected invalid, but valid\n", p)
		}
	}
}
