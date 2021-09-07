// Copyright 2019 The Scriggo Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"testing"
)

var validModulePaths = []string{
	"a", "A", "5", "c/e-g/x_y", "~", "a.b", "a..b", "con2",
	"A~B.txt", "A~B2"}

var invalidModulePaths = []string{
	"", ".", "-", "/", "/a", "a/", ".a/b", "c/d.", "!", "a//a",
	"~1", "SHORT~5.txt", "coN", "AUX.c", "Com2.txt", "LPT0"}

func TestValidModulePath(t *testing.T) {
	for _, p := range validModulePaths {
		if !validModulePath(p) {
			t.Errorf("path: %q, expected valid, but invalid\n", p)
		}
	}
	for _, p := range invalidModulePaths {
		if validModulePath(p) {
			t.Errorf("path: %q, expected invalid, but valid\n", p)
		}
	}
}

var validPackagePaths = []string{"a", "/a", "a/b", "/a/b", "../a", "../../a"}
var invalidPackagePaths = []string{"", ".", "..", "/", "a/", "//", "a//", "//a", "/..", "a/.."}

func TestValidTemplatePath(t *testing.T) {
	for _, p := range validPackagePaths {
		if !ValidTemplatePath(p) {
			t.Errorf("path: %q, expected valid, but invalid\n", p)
		}
	}
	for _, p := range invalidPackagePaths {
		if ValidTemplatePath(p) {
			t.Errorf("path: %q, expected invalid, but valid\n", p)
		}
	}
}
