// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"reflect"
	"strings"
	"testing"
)

func Test_splitErrorFromLine(t *testing.T) {
	cases := map[string]string{
		"x = 10 // ERROR `undefined: x`": "undefined: x",
		`f() // ERROR "cannot call f"`:   "cannot call f",
		`f() // a comment`:               "",
	}
	for line, expected := range cases {
		got := splitErrorFromLine(line)
		if got != expected {
			t.Errorf("line: '%s': expecting '%s', got '%s'", line, expected, got)
		}
	}
}

func Test_linesWithError(t *testing.T) {
	cases := []struct {
		src      []string
		expected []int
	}{
		{
			src: []string{
				"line 1",
				"line 2",
				"line 3",
			},
			expected: []int{},
		},
		{
			src: []string{
				"line 1",
				"line 2 // ERROR `something`",
				"line 3",
			},
			expected: []int{2},
		},
		{
			src: []string{
				"line 1",
				"line 2 // ERROR `something`",
				"line 3 // ERROR `something else`",
			},
			expected: []int{2, 3},
		},
	}
	for _, cas := range cases {
		src := strings.Join(cas.src, "\n")
		got := linesWithError(src)
		if !reflect.DeepEqual(cas.expected, got) {
			t.Errorf("src: %q: expecting %v, got %v", src, cas.expected, got)
		}
	}
}

func Test_differentiateSources(t *testing.T) {
	cases := []struct {
		src      string
		expected []errorcheckTest
	}{
		{
			src: joinLines([]string{
				`line 1`,
				`line 2`,
				`line 3`,
			}),
			expected: []errorcheckTest{},
		},
		{
			src: joinLines([]string{
				`line 1`,
				`line 2 // ERROR "something"`,
				`line 3`,
			}),
			expected: []errorcheckTest{
				errorcheckTest{
					src: joinLines([]string{
						`line 1`,
						`line 2 // ERROR "something"`,
						`line 3`,
					}),
					err: "something",
				},
			},
		},
		{
			src: joinLines([]string{
				`line 1`,
				`line 2 // ERROR "something"`,
				`line 3 // ERROR "something else"`,
			}),
			expected: []errorcheckTest{
				errorcheckTest{
					src: joinLines([]string{
						`line 1`,
						`line 2 // ERROR "something"`,
					}),
					err: "something",
				},
				errorcheckTest{
					src: joinLines([]string{
						`line 1`,
						`line 3 // ERROR "something else"`,
					}),
					err: "something else",
				},
			},
		},
		{
			src: joinLines([]string{
				`line 1`,
				`line 2 // ERROR "something"`,
				`line 3 // ERROR "something else"`,
				`line 4`,
				`line 5 // ERROR "another error message"`,
			}),
			expected: []errorcheckTest{
				errorcheckTest{
					src: joinLines([]string{
						`line 1`,
						`line 2 // ERROR "something"`,
						`line 4`,
					}),
					err: "something",
				},
				errorcheckTest{
					src: joinLines([]string{
						`line 1`,
						`line 3 // ERROR "something else"`,
						`line 4`,
					}),
					err: "something else",
				},
				errorcheckTest{
					src: joinLines([]string{
						`line 1`,
						`line 4`,
						`line 5 // ERROR "another error message"`,
					}),
					err: "another error message",
				},
			},
		},
	}
	for _, cas := range cases {
		expected := cas.expected
		got := differentiateSources(cas.src)
		if len(expected) != len(got) {
			t.Errorf("expecting len %d, got %d", len(expected), len(got))
			continue
		}
		for i := range expected {
			if expected[i].err != got[i].err {
				t.Errorf("expecting error '%s', got '%s'", expected[i].err, got[i].err)
				continue
			}
			if expected[i].src != got[i].src {
				t.Errorf("expected src:\n-------------------\n%s\n-------------------\n\ngot\n-------------------\n%s", expected[i].src, got[i].src)
				continue
			}
		}
		if !reflect.DeepEqual(expected, got) {
			panic("unexpected")
		}
	}
}

func joinLines(ss []string) string {
	ws := strings.Builder{}
	for _, s := range ss {
		ws.WriteString(s + "\n")
	}
	return ws.String()
}

func Test_removePrefixFromError(t *testing.T) {
	cases := map[string]string{
		`error`: `error`,
		`./prog.go:8:34: syntax error: unexpected s at end of statement`: `syntax error: unexpected s at end of statement`,
		`:8:34: syntax error: something`:                                 `syntax error: something`,
		`a:0:0: an error`:                                                `an error`,
	}
	for err, expected := range cases {
		got := removePrefixFromError(err)
		if expected != got {
			t.Errorf("'%s': expecting '%s', got '%s'", err, expected, got)
		}
	}
}
