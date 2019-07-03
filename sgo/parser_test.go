// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"reflect"
	"testing"
)

func Test_parseFileComment_error(t *testing.T) {
	cases := map[string]string{
		`//scriggo:`: `specify what to do`,
		`//scriggo: interpreters variable:"pkgs"`: `cannot use variable with interpreters`,
		`//scriggo: interpreters embedded`:        `cannot use embedded with interpreters`,
	}
	for input, expected := range cases {
		t.Run(input, func(t *testing.T) {
			_, got := parseFileComment(input)
			if got == nil {
				t.Fatalf("%s: expected error %q, got nothing", input, expected)
			}
			if got.Error() != expected {
				t.Fatalf("%s: expected error %q, got %q", input, expected, got.Error())
			}
		})
	}
}

func Test_parseFileComment(t *testing.T) {
	cases := map[string]fileComment{
		`//scriggo: embedded goos:"linux,darwin"`:           fileComment{embedded: true, goos: []string{"linux", "darwin"}},
		`//scriggo: embedded output:"/path/"`:               fileComment{embedded: true, output: "/path/"},
		`//scriggo: embedded variable:"pkgs"`:               fileComment{embedded: true, varName: "pkgs"},
		`//scriggo: embedded`:                               fileComment{embedded: true},
		`//scriggo: goos:"windows" embedded`:                fileComment{embedded: true, goos: []string{"windows"}},
		`//scriggo: interpreters:"program"`:                 fileComment{program: true},
		`//scriggo: interpreters:"script,program"`:          fileComment{script: true, program: true},
		`//scriggo: interpreters:"script,template,program"`: fileComment{script: true, program: true, template: true},
		`//scriggo: interpreters:"script"`:                  fileComment{script: true},
		`//scriggo: interpreters:"template"`:                fileComment{template: true},
		`//scriggo: interpreters`:                           fileComment{template: true, script: true, program: true},
	}
	for comment, want := range cases {
		t.Run(comment, func(t *testing.T) {
			got, err := parseFileComment(comment)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("comment: %s: wanted %#v, got %#v", comment, want, got)
			}
		})
	}
}

func Test_parseImportComment_error(t *testing.T) {
	cases := map[string]string{
		"//scriggo: uncapitalize": "cannot use option uncapitalize without option main",
	}
	for input, expected := range cases {
		t.Run(input, func(t *testing.T) {
			_, got := parseImportComment(input)
			if got == nil {
				t.Fatalf("%s: expected error %q, got nothing", input, expected)
			}
			if got.Error() != expected {
				t.Fatalf("%s: expected error %q, got %q", input, expected, got.Error())
			}
		})
	}
}

func Test_parseImportComment(t *testing.T) {
	cases := map[string]importComment{
		"//scriggo:":                   importComment{},
		`//scriggo: main`:              importComment{main: true},
		`//scriggo: main uncapitalize`: importComment{main: true, uncapitalize: true},
		`//scriggo: export:"Sleep"`: importComment{
			export: []string{"Sleep"},
		},
		`//scriggo: main export:"Sleep"`: importComment{
			main:   true,
			export: []string{"Sleep"},
		},
		`//scriggo: main uncapitalize export:"Sleep"`: importComment{
			main:         true,
			uncapitalize: true,
			export:       []string{"Sleep"},
		},
		`//scriggo: export:"Sleep,Duration"`: importComment{
			export: []string{"Sleep", "Duration"},
		},
		`//scriggo: main uncapitalize notexport:"Sleep"`: importComment{
			main:         true,
			uncapitalize: true,
			notexport:    []string{"Sleep"},
		},
		`//scriggo: notexport:"Sleep,Duration"`: importComment{
			notexport: []string{"Sleep", "Duration"},
		},
		`//scriggo: path:"test"`: importComment{
			newPath: "test",
			newName: "test",
		},
		`//scriggo: path:"newpath" export:"Sleep"`: importComment{
			export:  []string{"Sleep"},
			newPath: "newpath",
			newName: "newpath",
		},
		`//scriggo: export:"Sleep" path:"path/to/pkg"`: importComment{
			export:  []string{"Sleep"},
			newPath: "path/to/pkg",
			newName: "pkg",
		},
		`//scriggo: export:"Sleep" path:"path/to/test"`: importComment{
			export:  []string{"Sleep"},
			newPath: "path/to/test",
			newName: "test",
		},
	}
	for comment, want := range cases {
		t.Run(comment, func(t *testing.T) {
			got, err := parseImportComment(comment)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("comment: %s: wanted %#v, got %#v", comment, want, got)
			}
		})
	}
}

func Test_parse(t *testing.T) {
	cases := []struct {
		str  string
		opts []option
		kvs  []keyValues
	}{
		{
			str: "",
		},
		{
			str:  "option",
			opts: []option{"option"},
		},
		{
			str:  "option1 option2",
			opts: []option{"option1", "option2"},
		},
		{
			str: "key:value",
			kvs: []keyValues{
				keyValues{Key: "key", Values: []string{"value"}},
			},
		},
		{
			str: `option key:value option2`,
			opts: []option{
				`option`, `option2`,
			},
			kvs: []keyValues{
				keyValues{Key: `key`, Values: []string{"value"}},
			},
		},
		{
			str: `option key:"value" option2`,
			opts: []option{
				`option`, `option2`,
			},
			kvs: []keyValues{
				keyValues{Key: `key`, Values: []string{"value"}},
			},
		},
		{
			str: `key:value1,value2,value3`,
			kvs: []keyValues{
				keyValues{Key: `key`, Values: []string{`value1`, `value2`, `value3`}},
			},
		},
		{
			str: `key:"value1,value2,value3"`,
			kvs: []keyValues{
				keyValues{Key: `key`, Values: []string{`value1`, `value2`, `value3`}},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.str, func(t *testing.T) {
			gotOpts, gotKvs, err := parse(c.str)
			if err != nil {
				t.Fatal(err)
			}
			if len(c.opts) == 0 {
				c.opts = []option{}
			}
			if len(c.kvs) == 0 {
				c.kvs = []keyValues{}
			}
			if len(gotOpts) == 0 {
				gotOpts = []option{}
			}
			if len(gotKvs) == 0 {
				gotKvs = []keyValues{}
			}
			if !reflect.DeepEqual(gotOpts, c.opts) || !reflect.DeepEqual(gotKvs, c.kvs) {
				t.Fatalf("input: %q: expected %#v and %#v, got %#v and %#v", c.str, c.opts, c.kvs, gotOpts, gotKvs)
			}
		})
	}
}

func Test_tokenize(t *testing.T) {
	cases := map[string][]string{
		``:                                       []string{},
		`   `:                                    []string{},
		`word`:                                   []string{`word`},
		`word1 word2`:                            []string{`word1`, `word2`},
		`word1     word2`:                        []string{`word1`, `word2`},
		`key:value`:                              []string{`key`, `:`, `value`},
		`key:value1,value2`:                      []string{`key`, `:`, `value1,value2`},
		`key:"value1,value2"`:                    []string{`key`, `:`, `value1,value2`},
		`key:value1, word1`:                      []string{`key`, `:`, `value1,`, `word1`},
		`  key:value1,       word1  `:            []string{`key`, `:`, `value1,`, `word1`},
		`key:"value1, value2"`:                   []string{`key`, `:`, `value1, value2`},
		`word key:value`:                         []string{`word`, `key`, `:`, `value`},
		`word key:value1,value2 word2 key:value`: []string{`word`, `key`, `:`, `value1,value2`, `word2`, `key`, `:`, `value`},
		`word key:"value1,value2" word2 key:value`: []string{`word`, `key`, `:`, `value1,value2`, `word2`, `key`, `:`, `value`},
	}
	for input, expected := range cases {
		got, err := tokenize(input)
		if err != nil {
			t.Errorf("input: %q: %s", input, err.Error())
			continue
		}
		if !reflect.DeepEqual(expected, got) {
			t.Errorf("input: %q, expected: %#v, got %#v", input, expected, got)
		}
	}
}

func Test_tokenize_error(t *testing.T) {
	cases := map[string]string{
		`word:`:            `unexpected EOL after colon, expecting quote or word`,
		`word"`:            `unexpected quote after word`,
		`word1 "word2`:     `unexpected quote after word1`,
		`word1 key:"word2`: `unexpected EOL, expecting quote`,
		`:`:                `unexpected colon at beginning of line, expecting word or space`,
		`key: value`:       `unexpected space after colon`,
	}
	for input, expected := range cases {
		_, got := tokenize(input)
		if got == nil {
			t.Errorf("input: '%s': expecting error '%s', got nothing", input, expected)
			continue
		}
		if got.Error() != expected {
			t.Errorf("input: '%s': expecting error '%s', got '%s'", input, expected, got)
		}
	}
}
