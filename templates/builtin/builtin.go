// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package builtin provides simple functions that can be used as global
// functions in a Scriggo template.
//
// For example, to use the Min and Max functions as global min and max
// functions
//
//    globals := templates.Declarations{
//        "min": builtin.Min,
//        "max": builtin.Max,
//    }
//    opts := &templates.BuildOptions{
//        Globals: globals,
//    }
//    template, err := scriggoTemplates.Build(fsys, file, opts)
//
// And to use them in a template
//
//    {{ min(x, y) }}
//    {{ max(x, y) }}
//
// Use this Declarations value to use all the builtin of this package
// in a template
//
//    templates.Declarations{
//        "abbreviate":    builtin.Abbreviate,
//        "abs":           builtin.Abs,
//        "base64":        builtin.Base64,
//        "capitalize":    builtin.Capitalize,
//        "capitalizeAll": builtin.CapitalizeAll,
//        "hasPrefix":     builtin.HasPrefix,
//        "hasSuffix":     builtin.HasSuffix,
//        "hex":           builtin.Hex,
//        "hmacSHA1":      builtin.HmacSHA1,
//        "hmacSHA256":    builtin.HmacSHA256,
//        "htmlEscape":    builtin.HtmlEscape,
//        "index":         builtin.Index,
//        "indexAny":      builtin.IndexAny,
//        "join":          builtin.Join,
//        "lastIndex":     builtin.LastIndex,
//        "max":           builtin.Max,
//        "md5":           builtin.Md5,
//        "min":           builtin.Min,
//        "queryEscape":   builtin.QueryEscape,
//        "replace":       builtin.Replace,
//        "replaceAll":    builtin.ReplaceAll,
//        "reverse":       builtin.Reverse,
//        "runeCount":     builtin.RuneCount,
//        "sha1":          builtin.Sha1,
//        "sha256":        builtin.Sha256,
//        "sort":          builtin.Sort,
//        "split":         builtin.Split,
//        "splitAfter":    builtin.SplitAfter,
//        "splitAfterN":   builtin.SplitAfterN,
//        "splitN":        builtin.SplitN,
//        "sprint":        builtin.Sprint,
//        "sprintf":       builtin.Sprintf,
//        "toKebab":       builtin.ToKebab,
//        "toLower":       builtin.ToLower,
//        "toUpper":       builtin.ToUpper,
//        "trim":          builtin.Trim,
//        "trimLeft":      builtin.TrimLeft,
//        "trimPrefix":    builtin.TrimPrefix,
//        "trimRight":     builtin.TrimRight,
//        "trimSuffix":    builtin.TrimSuffix,
//    }
//
package builtin

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/open2b/scriggo/templates"
)

// Abbreviate abbreviates s to almost n runes. If s is longer than n runes,
// the abbreviated string terminates with "...".
func Abbreviate(s string, n int) string {
	const spaces = " \n\r\t\f" // https://infra.spec.whatwg.org/#ascii-whitespace
	s = strings.TrimRight(s, spaces)
	if len(s) <= n {
		return s
	}
	if n < 3 {
		return ""
	}
	p := 0
	n2 := 0
	for i := range s {
		switch p {
		case n - 2:
			n2 = i
		case n:
			break
		}
		p++
	}
	if p < n {
		return s
	}
	if p = strings.LastIndexAny(s[:n2], spaces); p > 0 {
		s = strings.TrimRight(s[:p], spaces)
	} else {
		s = ""
	}
	if l := len(s) - 1; l >= 0 && (s[l] == '.' || s[l] == ',') {
		s = s[:l]
	}
	return s + "..."
}

// Abs returns the absolute value of x.
func Abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// Base64 returns the base64 encoding of s.
func Base64(s string) string {
	if s == "" {
		return s
	}
	return base64.StdEncoding.EncodeToString([]byte(s))
}

// Capitalize returns a copy of the string src with the first non-separator in
// upper case.
func Capitalize(src string) string {
	for i, r := range src {
		if isSeparator(r) {
			continue
		}
		if unicode.IsUpper(r) {
			return src
		}
		r = unicode.ToUpper(r)
		b := strings.Builder{}
		b.Grow(len(src))
		b.WriteString(src[:i])
		b.WriteRune(r)
		b.WriteString(src[i+utf8.RuneLen(r):])
		return b.String()
	}
	return src
}

// CapitalizeAll returns a copy of the string src with the first letter of each
// word in upper case.
func CapitalizeAll(src string) string {
	prev := ' '
	return strings.Map(func(r rune) rune {
		if isSeparator(prev) {
			prev = r
			return unicode.ToUpper(r)
		}
		prev = r
		return r
	}, src)
}

// HasPrefix tests whether the string s begins with prefix.
func HasPrefix(s, prefix string) bool {
	return strings.HasPrefix(s, prefix)
}

// HasSuffix tests whether the string s ends with suffix.
func HasSuffix(s, suffix string) bool {
	return strings.HasPrefix(s, suffix)
}

// Hex returns the hexadecimal encoding of src.
func Hex(src string) string {
	if src == "" {
		return src
	}
	return hex.EncodeToString([]byte(src))
}

// HmacSHA1 returns the HMAC-SHA1 tag for the given message and key, as a
// base64 encoded string.
func HmacSHA1(message, key string) string {
	mac := hmac.New(sha1.New, []byte(key))
	_, _ = io.WriteString(mac, message)
	s := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return s
}

// HmacSHA256 returns the HMAC-SHA256 tag for the given message and key, as a
// base64 encoded string.
func HmacSHA256(message, key string) string {
	mac := hmac.New(sha256.New, []byte(key))
	_, _ = io.WriteString(mac, message)
	s := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return s
}

// HtmlEscape escapes s, replacing the characters <, >, &, " and ' and returns
// the escaped string as templates.HTML type.
func HtmlEscape(s string) templates.HTML {
	return templates.HTMLEscape(s)
}

// Index returns the index of the first instance of substr in s, or -1 if substr is not present in s.
func Index(s, substr string) int {
	return strings.Index(s, substr)
}

// IndexAny returns the index of the first instance of any Unicode code point
// from chars in s, or -1 if no Unicode code point from chars is present in s.
func IndexAny(s, chars string) int {
	return strings.IndexAny(s, chars)
}

// Join concatenates the elements of its first argument to create a single string. The separator
// string sep is placed between elements in the resulting string.
func Join(elems []string, sep string) string {
	return strings.Join(elems, sep)
}

// LastIndex returns the index of the last instance of substr in s, or -1 if substr is not present in s.
func LastIndex(s, substr string) int {
	return strings.LastIndex(s, substr)
}

// Max returns the larger of x or y.
func Max(x, y int) int {
	if x < y {
		return y
	}
	return x
}

// Md5 returns the MD5 checksum of src as an hexadecimal encoded string.
func Md5(src string) string {
	h := md5.New()
	h.Write([]byte(src))
	return hex.EncodeToString(h.Sum(nil))
}

// Min returns the smaller of x or y.
func Min(x, y int) int {
	if y < x {
		return y
	}
	return x
}

// QueryEscape escapes the string so it can be safely placed
// inside a URL query.
func QueryEscape(s string) string {
	const hexchars = "0123456789abcdef"
	last := 0
	numHex := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if '0' <= c && c <= '9' || 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' ||
			c == '-' || c == '.' || c == '_' {
			continue
		}
		last = i + 1
		numHex++
	}
	if numHex == 0 {
		return s
	}
	// Fill buffer.
	j := 0
	b := make([]byte, len(s)+2*numHex)
	for i := 0; i < last; i++ {
		c := s[i]
		if '0' <= c && c <= '9' || 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' ||
			c == '-' || c == '.' || c == '_' {
			b[j] = c
		} else {
			b[j] = '%'
			j++
			b[j] = hexchars[c>>4]
			j++
			b[j] = hexchars[c&0xF]
		}
		j++
	}
	if j != len(b) {
		copy(b[j:], s[last:])
	}
	return string(b)
}

// Replace returns a copy of the string s with the first n
// non-overlapping instances of old replaced by new.
// If old is empty, it matches at the beginning of the string
// and after each UTF-8 sequence, yielding up to k+1 replacements
// for a k-rune string.
// If n < 0, there is no limit on the number of replacements.
func Replace(s, old, new string, n int) string {
	return strings.Replace(s, old, new, n)
}

// ReplaceAll returns a copy of the string s with all
// non-overlapping instances of old replaced by new.
// If old is empty, it matches at the beginning of the string
// and after each UTF-8 sequence, yielding up to k+1 replacements
// for a k-rune string.
func ReplaceAll(s, old, new string) string {
	return strings.ReplaceAll(s, old, new)
}

var errNoSlice = errors.New("no slice")

// Reverse returns the reverse order for data.
func Reverse(data interface{}) {
	if data == nil {
		return
	}
	rv := reflect.ValueOf(data)
	if rv.Kind() != reflect.Slice {
		panic(errNoSlice)
	}
	l := rv.Len()
	if l <= 1 {
		return
	}
	swap := reflect.Swapper(data)
	for i, j := 0, l-1; i < j; i, j = i+1, j-1 {
		swap(i, j)
	}
	return
}

// RuneCount returns the number of runes in s. Erroneous and short
// encodings are treated as single runes of width 1 byte.
func RuneCount(s string) (n int) {
	return utf8.RuneCountInString(s)
}

// Sha1 returns the SHA1 checksum of src as an hexadecimal encoded string.
func Sha1(src string) string {
	h := sha1.New()
	h.Write([]byte(src))
	return hex.EncodeToString(h.Sum(nil))
}

// Sha256 returns the SHA256 checksum of src as an hexadecimal encoded string.
func Sha256(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

// Sort sorts the provided slice given the provided less function.
//
// The function panics if the provided interface is not a slice.
func Sort(slice interface{}, less func(i, j int) bool) {
	if slice == nil {
		return
	}
	if less != nil {
		sort.Slice(slice, less)
		return
	}
	// no reflect
	switch s := slice.(type) {
	case nil:
	case []string:
		sort.Strings(s)
	case []rune:
		sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	case []byte:
		sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	case []int:
		sort.Ints(s)
	case []float64:
		sort.Float64s(s)
	case []templates.HTML:
		sort.Slice(s, func(i, j int) bool { return string(s[i]) < string(s[j]) })
	case []templates.CSS:
		sort.Slice(s, func(i, j int) bool { return string(s[i]) < string(s[j]) })
	case []templates.JS:
		sort.Slice(s, func(i, j int) bool { return string(s[i]) < string(s[j]) })
	case []templates.JSON:
		sort.Slice(s, func(i, j int) bool { return string(s[i]) < string(s[j]) })
	case []templates.Markdown:
		sort.Slice(s, func(i, j int) bool { return string(s[i]) < string(s[j]) })
	}
	// reflect
	sortSlice(slice)
}

// Split slices s into all substrings separated by sep and returns a slice of
// the substrings between those separators.
//
// If s does not contain sep and sep is not empty, Split returns a
// slice of length 1 whose only element is s.
//
// If sep is empty, Split splits after each UTF-8 sequence. If both s
// and sep are empty, Split returns an empty slice.
//
// It is equivalent to SplitN with a count of -1.
func Split(s, sep string) []string {
	return strings.Split(s, sep)
}

// SplitAfter slices s into all substrings after each instance of sep and
// returns a slice of those substrings.
//
// If s does not contain sep and sep is not empty, SplitAfter returns
// a slice of length 1 whose only element is s.
//
// If sep is empty, SplitAfter splits after each UTF-8 sequence. If
// both s and sep are empty, SplitAfter returns an empty slice.
//
// It is equivalent to SplitAfterN with a count of -1.
func SplitAfter(s, sep string) []string {
	return strings.SplitAfter(s, sep)
}

// SplitAfterN slices s into substrings after each instance of sep and
// returns a slice of those substrings.
//
// The count determines the number of substrings to return:
//   n > 0: at most n substrings; the last substring will be the unsplit remainder.
//   n == 0: the result is nil (zero substrings)
//   n < 0: all substrings
//
// Edge cases for s and sep (for example, empty strings) are handled
// as described in the documentation for SplitAfter.
func SplitAfterN(s, sep string, n int) []string {
	return strings.SplitAfterN(s, sep, n)
}

// SplitN slices s into substrings separated by sep and returns a slice of
// the substrings between those separators.
//
// The count determines the number of substrings to return:
//   n > 0: at most n substrings; the last substring will be the unsplit remainder.
//   n == 0: the result is nil (zero substrings)
//   n < 0: all substrings
//
// Edge cases for s and sep (for example, empty strings) are handled
// as described in the documentation for Split.
func SplitN(s, sep string, n int) []string {
	return strings.SplitN(s, sep, n)
}

// Sprint formats using the default formats for its operands and returns the resulting string.
// Spaces are added between operands when neither is a string.
func Sprint(a ...interface{}) string {
	return fmt.Sprint(a...)
}

// Sprintf formats according to a format specifier and returns the resulting string.
func Sprintf(format string, a ...interface{}) string {
	return fmt.Sprintf(format, a...)
}

// ToKebab returns a copy of the string s in kebab case form.
func ToKebab(s string) string {
	b := strings.Builder{}
	b.Grow(len(s) + 2) // optimize for a string with 3 words.
	noDash := false    // true if the last written rune is not a dash.
	runes := []rune(s)
	n := len(runes)
	for i := 0; i < n; i++ {
		r := runes[i]
		switch {
		case unicode.IsLower(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			noDash = true
		case unicode.IsUpper(r):
			if noDash && (unicode.IsLower(runes[i-1]) || i+1 < n && unicode.IsLower(runes[i+1])) {
				b.WriteByte('-')
			}
			b.WriteRune(unicode.ToLower(r))
			noDash = true
		default:
			if noDash && i+1 < n {
				b.WriteByte('-')
				noDash = false
			}
		}
	}
	return strings.TrimSuffix(b.String(), "-")
}

// ToLower returns s with all Unicode letters mapped to their lower case.
func ToLower(s string) string {
	return strings.ToLower(s)
}

// ToUpper returns s with all Unicode letters mapped to their upper case.
func ToUpper(s string) string {
	return strings.ToUpper(s)
}

// Trim returns a slice of the string s with all leading and
// trailing Unicode code points contained in cutset removed.
func Trim(s, cutset string) string {
	return strings.Trim(s, cutset)
}

// TrimLeft returns a slice of the string s with all leading
// Unicode code points contained in cutset removed.
//
// To remove a prefix, use TrimPrefix instead.
func TrimLeft(s, cutset string) string {
	return strings.TrimLeft(s, cutset)
}

// TrimPrefix returns s without the provided leading prefix string.
// If s doesn't start with prefix, s is returned unchanged.
func TrimPrefix(s, prefix string) string {
	return strings.TrimPrefix(s, prefix)
}

// TrimRight returns a slice of the string s, with all trailing
// Unicode code points contained in cutset removed.
//
// To remove a suffix, use TrimSuffix instead.
func TrimRight(s, cutset string) string {
	return strings.TrimRight(s, cutset)
}

// TrimSuffix returns s without the provided trailing suffix string.
// If s doesn't end with suffix, s is returned unchanged.
func TrimSuffix(s, suffix string) string {
	return strings.TrimSuffix(s, suffix)
}

// isSeparator reports whether the rune could mark a word boundary.
// TODO: update when package unicode captures more of the properties.
func isSeparator(r rune) bool {

	// The code of this function is taken from the Go standard library and
	// is copyright of the Go Authors.

	// ASCII alphanumerics and underscore are not separators
	if r <= 0x7F {
		switch {
		case '0' <= r && r <= '9':
			return false
		case 'a' <= r && r <= 'z':
			return false
		case 'A' <= r && r <= 'Z':
			return false
		case r == '_':
			return false
		}
		return true
	}
	// Letters and digits are not separators
	if unicode.IsLetter(r) || unicode.IsDigit(r) {
		return false
	}
	// Otherwise, all we can do for now is treat spaces as separators.
	return unicode.IsSpace(r)
}
