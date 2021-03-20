// Copyright (c) 2021 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package builtin

import (
	"regexp"
)

// Regexp represents a regular expression.
type Regexp struct {
	r *regexp.Regexp
}

// Match reports whether the string s contains any match of the regular
// expression.
func (re Regexp) Match(s string) bool {
	return re.r.MatchString(s)
}

// Find returns a string holding the text of the leftmost match in s of the
// regular expression. If there is no match, the return value is an empty
// string, but it will also be empty if the regular expression successfully
// matches an empty string. Use FindSubmatch if it is necessary to distinguish
// these cases.
func (re Regexp) Find(s string) string {
	return re.r.FindString(s)
}

// FindAll is the 'All' version of Find; it returns a slice of all successive
// matches of the expression, as defined by the 'All' description in the Go
// regexp package comment. A return value of nil indicates no match.
func (re Regexp) FindAll(s string, n int) []string {
	return re.r.FindAllString(s, n)
}

// FindAllSubmatch is the 'All' version of FindSubmatch; it returns a slice of
// all successive matches of the expression, as defined by the 'All'
// description in the Go regexp package comment. A return value of nil
// indicates no match.
func (re Regexp) FindAllSubmatch(s string, n int) [][]string {
	return re.r.FindAllStringSubmatch(s, n)
}

// FindSubmatch returns a slice of strings holding the text of the leftmost
// match of the regular expression in s and the matches, if any, of its
// subexpressions, as defined by the 'Submatch' description in the Go regexp
// package comment. A return value of nil indicates no match.
func (re Regexp) FindSubmatch(s string) []string {
	return re.r.FindStringSubmatch(s)
}

// ReplaceAll returns a copy of src, replacing matches of the Regexp with the
// replacement string repl. Inside repl, $ signs are interpreted as in Expand
// method of the Go regexp package, so for instance $1 represents the text of
// the first submatch.
func (re Regexp) ReplaceAll(src, repl string) string {
	return re.r.ReplaceAllString(src, repl)
}

// ReplaceAllFunc returns a copy of src in which all matches of the Regexp
// have been replaced by the return value of function repl applied to the
// matched byte slice. The replacement returned by repl is substituted
// directly, without using expanding.
func (re Regexp) ReplaceAllFunc(src string, repl func(string) string) string {
	return re.r.ReplaceAllStringFunc(src, repl)
}

// Split slices s into substrings separated by the expression and returns a
// slice of the substrings between those expression matches.
//
// The slice returned by this method consists of all the substrings of s not
// contained in the slice returned by FindAllString. When called on an
// expression that contains no metacharacters, it is equivalent to SplitN.
//
// The count determines the number of substrings to return:
//   n > 0: at most n substrings; the last substring will be the unsplit remainder.
//   n == 0: the result is nil (zero substrings)
//   n < 0: all substrings
func (re Regexp) Split(s string, n int) []string {
	return re.r.Split(s, n)
}
