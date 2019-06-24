// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"reflect"
	"testing"
)

func Test_parseCommentTag(t *testing.T) {
	tests := map[string]commentTag{
		"//scriggo:":                   commentTag{},
		`//scriggo: main`:              commentTag{main: true},
		`//scriggo: main uncapitalize`: commentTag{main: true, uncapitalize: true},
		`//scriggo: export:"Sleep"`: commentTag{
			export: []string{"Sleep"},
		},
		`//scriggo: main export:"Sleep"`: commentTag{
			main:   true,
			export: []string{"Sleep"},
		},
		`//scriggo: main uncapitalize export:"Sleep"`: commentTag{
			main:         true,
			uncapitalize: true,
			export:       []string{"Sleep"},
		},
		`//scriggo: export:"Sleep,Duration"`: commentTag{
			export: []string{"Sleep", "Duration"},
		},
		`//scriggo: main uncapitalize notexport:"Sleep"`: commentTag{
			main:         true,
			uncapitalize: true,
			notexport:    []string{"Sleep"},
		},
		`//scriggo: notexport:"Sleep,Duration"`: commentTag{
			notexport: []string{"Sleep", "Duration"},
		},
		`//scriggo: main uncapitalize path:"test"`: commentTag{
			main:         true,
			uncapitalize: true,
			path:         "test",
			pkgName:      "test",
		},
		`//scriggo: path:"newpath" export:"Sleep"`: commentTag{
			export:  []string{"Sleep"},
			path:    "newpath",
			pkgName: "newpath",
		},
		`//scriggo: main export:"Sleep" path:"path/to/pkg"`: commentTag{
			main:    true,
			export:  []string{"Sleep"},
			path:    "path/to/pkg",
			pkgName: "pkg",
		},
		`//scriggo: main uncapitalize export:"Sleep" path:"path/to/test"`: commentTag{
			main:         true,
			uncapitalize: true,
			export:       []string{"Sleep"},
			path:         "path/to/test",
			pkgName:      "test",
		},
	}
	for comment, want := range tests {
		t.Run(comment, func(t *testing.T) {
			got, err := parseCommentTag(comment)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("wanted %+v, got %+v", want, got)
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
	}
	for _, tt := range tests {
		t.Run(tt.current, func(t *testing.T) {
			if got := goBaseVersion(tt.current); got != tt.want {
				t.Errorf("goBaseVersion(%s) = %v, want %v", tt.current, got, tt.want)
			}
		})
	}
}

func Test_filterIncluding(t *testing.T) {
	cases := []struct {
		decls    map[string]string
		include  []string
		expected map[string]string
	}{
		{
			decls: map[string]string{
				"A": "a",
				"B": "b",
				"C": "c",
			},
			include: []string{"A"},
			expected: map[string]string{
				"A": "a",
			},
		},
		{
			decls: map[string]string{
				"A": "a",
				"B": "b",
				"C": "c",
			},
			include:  []string{},
			expected: map[string]string{},
		},
		{
			decls: map[string]string{
				"A": "a",
				"B": "b",
				"C": "c",
			},
			include: []string{"A", "C"},
			expected: map[string]string{
				"A": "a",
				"C": "c",
			},
		},
	}
	for _, c := range cases {
		got, err := filterIncluding(c.decls, c.include)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(c.expected, got) {
			t.Errorf("decls: %v, expected: %v, got: %v", c.decls, c.expected, got)
		}
	}
}

func Test_filterExcluding(t *testing.T) {
	cases := []struct {
		decls    map[string]string
		exclude  []string
		expected map[string]string
	}{
		{
			decls: map[string]string{
				"A": "a",
				"B": "b",
				"C": "c",
			},
			exclude: []string{"A"},
			expected: map[string]string{
				"B": "b",
				"C": "c",
			},
		},
		{
			decls: map[string]string{
				"A": "a",
				"B": "b",
				"C": "c",
			},
			exclude: []string{},
			expected: map[string]string{
				"A": "a",
				"B": "b",
				"C": "c",
			},
		},
		{
			decls: map[string]string{
				"A": "a",
				"B": "b",
				"C": "c",
			},
			exclude: []string{"A", "C"},
			expected: map[string]string{
				"B": "b",
			},
		},
	}
	for _, c := range cases {
		got, err := filterExcluding(c.decls, c.exclude)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(c.expected, got) {
			t.Errorf("decls: %v, expected: %v, got: %v", c.decls, c.expected, got)
		}
	}
}
