// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"testing"
)

func Test_uncapitalize(t *testing.T) {
	cases := map[string]string{
		"name":         "name",
		"Name":         "name",
		"ADSL":         "adsl",
		"ADSLAndOther": "adslAndOther",
		"DoubleWord":   "doubleWord",
		"X":            "x",
		"unExported":   "unExported",
		"AbC":          "abC",
		"Èident":       "èident",
		"È":            "è",
		"ÀÈÈ":          "àèè",
		"àÀÈÒò":        "àÀÈÒò",
	}
	for input, expected := range cases {
		t.Run(input, func(t *testing.T) {
			got := uncapitalize(input)
			if got != expected {
				t.Fatalf("input: %q, expected %q, got %q", input, expected, got)
			}
		})
	}
}

func Test_nextGoVersion(t *testing.T) {
	tests := []struct {
		current string
		want    string
	}{
		{
			current: "go1.12",
			want:    "go1.13",
		},
		{
			current: "go1.8",
			want:    "go1.9",
		},
		{
			current: "go1.20.1",
			want:    "go1.21",
		},
		{
			current: "go1.1000",
			want:    "go1.1001",
		},
		{
			current: "go1.1000beta2",
			want:    "go1.1001",
		},
		{
			current: "go1.13beta1",
			want:    "go1.14",
		},
	}
	for _, tt := range tests {
		t.Run(tt.current, func(t *testing.T) {
			if got := nextGoVersion(tt.current); got != tt.want {
				t.Errorf("nextGoVersion(%s) = %v, want %v", tt.current, got, tt.want)
			}
		})
	}
}

func Test_goBaseVersion(t *testing.T) {
	tests := []struct {
		current string
		want    string
	}{
		{
			current: "go1.12",
			want:    "go1.12",
		},
		{
			current: "go1.8",
			want:    "go1.8",
		},
		{
			current: "go1.20.1",
			want:    "go1.20",
		},
		{
			current: "go1.13beta1",
			want:    "go1.13",
		},
		{
			current: "go1.1000",
			want:    "go1.1000",
		},
		{
			current: "go1.1000beta5",
			want:    "go1.1000",
		},
		{
			current: "go1.13rc1",
			want:    "go1.13",
		},
	}
	for _, tt := range tests {
		t.Run(tt.current, func(t *testing.T) {
			if got := goBaseVersion(tt.current); got != tt.want {
				t.Errorf("goBaseVersion(%s) = %v, want %v", tt.current, got, tt.want)
			}
		})
	}
}

func Test_checkPackagePath(t *testing.T) {
	cases := map[string]string{
		`main`:          ``,
		`internal`:      `use of internal package "internal" not allowed`,
		`internal/pkg`:  `use of internal package "internal/pkg" not allowed`,
		`internal/main`: `use of internal package "internal/main" not allowed`,
		`fmt`:           ``,
		`?`:             `invalid path path "?"`,
	}
	for path, want := range cases {
		t.Run(path, func(t *testing.T) {
			got := checkPackagePath(path)
			switch {
			case want == "" && got == nil:
				// Ok.
			case want == "" && got != nil:
				t.Fatalf("path '%s': no error expected, got '%s'", path, got)
			case want != "" && got == nil:
				t.Fatalf("path '%s': error '%s' expected, got nothing", path, want)
			case want != "" && got != nil:
				if want == got.Error() {
					// Ok
				} else {
					t.Fatalf("path: '%s': expecting error '%s', got '%s'", path, want, got)
				}
			}
		})
	}
}

func Test_hasStdlibPrefix(t *testing.T) {
	cases := map[string]bool{
		`main`:         false,
		`fmt`:          true,
		`archive`:      true,
		`archiv`:       false,
		`archive/tar`:  true,
		`testing`:      true,
		`path/to/pkg`:  true,
		`path/to/test`: true,
	}
	for path, want := range cases {
		t.Run(path, func(t *testing.T) {
			got := hasStdlibPrefix(path)
			if got != want {
				t.Fatalf("path '%s': expecting %t, got %t", path, want, got)
			}
		})
	}
}
