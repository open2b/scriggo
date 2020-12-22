// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"testing"
)

var validPaths = []string{"a", "/a", "a/b", "/a/b", "../a", "../../a"}

var invalidPaths = []string{"", ".", "..", "/", "a/", "//", "a//", "//a", "/..", "a/.."}

func TestValidTemplatePath(t *testing.T) {
	for _, p := range validPaths {
		if !ValidTemplatePath(p) {
			t.Errorf("path: %q, expected valid, but invalid\n", p)
		}
	}
	for _, p := range invalidPaths {
		if ValidTemplatePath(p) {
			t.Errorf("path: %q, expected invalid, but valid\n", p)
		}
	}
}
