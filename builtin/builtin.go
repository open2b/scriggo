// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package builtin provides simple functions, types and constants that can be
// used as globals in a Scriggo template.
//
// For example, to use the Min and Max functions as global min and max
// functions
//
//    globals := native.Declarations{
//        "min": builtin.Min,
//        "max": builtin.Max,
//    }
//    opts := &scriggo.BuildOptions{
//        Globals: globals,
//    }
//    template, err := scriggo.BuildTemplate(fsys, file, opts)
//
// And to use them in a template
//
//    {{ min(x, y) }}
//    {{ max(x, y) }}
//
// Use the regexp function and the returned Regex value in this way
//
//    {% var re = regexp(`(scrig)go`) %}
//    {{ re.Match("go") }}
//    {{ re.Match("scriggo") }}
//
// Use this Declarations value to use all the builtin of this package in a
// template or choose the most appropriate
//
//  native.Declarations{
//  	// crypto
//  	"hmacSHA1":   builtin.HmacSHA1,
//  	"hmacSHA256": builtin.HmacSHA256,
//  	"sha1":       builtin.Sha1,
//  	"sha256":     builtin.Sha256,
//
//  	// encoding
//  	"base64":            builtin.Base64,
//  	"hex":               builtin.Hex,
//  	"marshalJSON":       builtin.MarshalJSON,
//  	"marshalJSONIndent": builtin.MarshalJSONIndent,
//  	"md5":               builtin.Md5,
//  	"unmarshalJSON":     builtin.UnmarshalJSON,
//
//  	// html
//  	"htmlEscape": builtin.HtmlEscape,
//
//  	// math
//  	"abs": builtin.Abs,
//  	"max": builtin.Max,
//  	"min": builtin.Min,
//
//  	// net
//  	"File":        reflect.TypeOf((*builtin.File)(nil)).Elem(),
//  	"FormData":    reflect.TypeOf(builtin.FormData{}),
//  	"form":        (*builtin.FormData)(nil),
//  	"queryEscape": builtin.QueryEscape,
//
//  	// regexp
//  	"Regexp": reflect.TypeOf(builtin.Regexp{}),
//  	"regexp": builtin.RegExp,
//
//  	// sort
//  	"reverse": builtin.Reverse,
//  	"sort":    builtin.Sort,
//
//  	// strconv
//  	"formatFloat": builtin.FormatFloat,
//  	"formatInt":   builtin.FormatInt,
//  	"parseFloat":  builtin.ParseFloat,
//  	"parseInt":    builtin.ParseInt,
//
//  	// strings
//  	"abbreviate":    builtin.Abbreviate,
//  	"capitalize":    builtin.Capitalize,
//  	"capitalizeAll": builtin.CapitalizeAll,
//  	"hasPrefix":     builtin.HasPrefix,
//  	"hasSuffix":     builtin.HasSuffix,
//  	"index":         builtin.Index,
//  	"indexAny":      builtin.IndexAny,
//  	"join":          builtin.Join,
//  	"lastIndex":     builtin.LastIndex,
//  	"replace":       builtin.Replace,
//  	"replaceAll":    builtin.ReplaceAll,
//  	"runeCount":     builtin.RuneCount,
//  	"split":         builtin.Split,
//  	"splitAfter":    builtin.SplitAfter,
//  	"splitAfterN":   builtin.SplitAfterN,
//  	"splitN":        builtin.SplitN,
//  	"sprint":        builtin.Sprint,
//  	"sprintf":       builtin.Sprintf,
//  	"toKebab":       builtin.ToKebab,
//  	"toLower":       builtin.ToLower,
//  	"toUpper":       builtin.ToUpper,
//  	"trim":          builtin.Trim,
//  	"trimLeft":      builtin.TrimLeft,
//  	"trimPrefix":    builtin.TrimPrefix,
//  	"trimRight":     builtin.TrimRight,
//  	"trimSuffix":    builtin.TrimSuffix,
//
//  	// time
//  	"Duration":      reflect.TypeOf(builtin.Duration(0)),
//  	"Hour":          time.Hour,
//  	"Microsecond":   time.Microsecond,
//  	"Millisecond":   time.Millisecond,
//  	"Minute":        time.Minute,
//  	"Nanosecond":    time.Nanosecond,
//  	"Second":        time.Second,
//  	"Time":          reflect.TypeOf(builtin.Time{}),
//  	"date":          builtin.Date,
//  	"now":           builtin.Now,
//  	"parseDuration": builtin.ParseDuration,
//  	"parseTime":     builtin.ParseTime,
//  	"unixTime":      builtin.UnixTime,
//
//  }
//
// To initialize the form builtin value, with data read from the request r,
// use this map as vars argument to Run
//
//    map[string]interface{}{"form": builtin.NewFormData(r, 10)}
//
package builtin

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/open2b/scriggo"
	"github.com/open2b/scriggo/native"
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

// CapitalizeAll returns a copy of the string src with the first letter of
// each word in upper case.
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

// Date returns the time corresponding to the given date with time zone
// determined by location. If location does not exist, it returns an error.
//
// For example, the following call returns March 27, 2021 11:21:14.964553705
// CET.
//   Date(2021, 3, 27, 11, 21, 14, 964553705, "Europe/Rome")
//
// For UTC use "" or "UTC" as location. For the system's local time zone use
// "Local" as location.
//
// The month, day, hour, min, sec and nsec values may be outside their usual
// ranges and will be normalized during the conversion. For example, October
// 32 converts to November 1.
//
func Date(year, month, day, hour, min, sec, nsec int, location string) (Time, error) {
	loc, err := time.LoadLocation(location)
	if err != nil {
		return Time{}, replacePrefix(err, "time", "date")
	}
	return NewTime(time.Date(year, time.Month(month), day, hour, min, sec, nsec, loc)), nil
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
// the escaped string as native.HTML type.
func HtmlEscape(s string) native.HTML {
	return scriggo.HTMLEscape(s)
}

// Index returns the index of the first instance of substr in s, or -1 if
// substr is not present in s.
func Index(s, substr string) int {
	return strings.Index(s, substr)
}

// IndexAny returns the index of the first instance of any Unicode code point
// from chars in s, or -1 if no Unicode code point from chars is present in s.
func IndexAny(s, chars string) int {
	return strings.IndexAny(s, chars)
}

// FormatInt returns the string representation of i in the given base, for
// 2 <= base <= 36. The result uses the lower-case letters 'a' to 'z' for
// digit values >= 10.
//
// It panics if base is not in the range.
func FormatInt(i int, base int) string {
	if base < 2 || base > 36 {
		panic("formatInt: invalid base " + strconv.Itoa(base))
	}
	return strconv.FormatInt(int64(i), base)
}

// FormatFloat converts the floating-point number f to a string, according to
// the given format and precision. It can round the result.
//
// The format is one of "e", "f" or "g"
//  "e": -d.dddde±dd, a decimal exponent
//  "f": -ddd.dddd, no exponent
//  "g": "e" for large exponents, "f" otherwises
//
// The precision, for -1 <= precision <= 1000, controls the number of digits
// (excluding the exponent). The special precision -1 uses the smallest number
// of digits necessary such that ParseFloat will return f exactly. For "e"
// and "f" it is the number of digits after the decimal point. For "g" it is
// the maximum number of significant digits (trailing zeros are removed).
//
// If the format or the precision is not valid, FormatFloat panics.
func FormatFloat(f float64, format string, precision int) string {
	switch format {
	case "e", "f", "g":
	default:
		panic("formatFloat: invalid format " + strconv.Quote(format))
	}
	if precision < -1 || precision > 1000 {
		panic("formatFloat: invalid precision " + strconv.Itoa(precision))
	}
	return strconv.FormatFloat(f, format[0], precision, 64)
}

// Join concatenates the elements of its first argument to create a single
// string. The separator string sep is placed between elements in the
// resulting string.
func Join(elems []string, sep string) string {
	return strings.Join(elems, sep)
}

// LastIndex returns the index of the last instance of substr in s, or -1 if
// substr is not present in s.
func LastIndex(s, substr string) int {
	return strings.LastIndex(s, substr)
}

// MarshalJSON returns the JSON encoding of v.
//
// See https://golang.org/pkg/encoding/json/#Marshal for details.
func MarshalJSON(v interface{}) (native.JSON, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", replacePrefix(err, "json", "marshalJSON")
	}
	return native.JSON(b), nil
}

// MarshalJSONIndent is like MarshalJSON but indents the output. Each JSON
// element in the output will begin on a new line beginning with prefix
// followed by one or more copies of indent according to the indentation
// nesting. prefix and indent can only contain whitespace: ' ', '\t', '\n' and
// '\r'.
func MarshalJSONIndent(v interface{}, prefix, indent string) (native.JSON, error) {
	if !onlyJSONWhitespace(prefix) {
		return "", errors.New("marshalJSONIndent: prefix does not contain only whitespace")
	}
	if !onlyJSONWhitespace(indent) {
		return "", errors.New("marshalJSONIndent: indent does not contain only whitespace")
	}
	b, err := json.MarshalIndent(v, prefix, indent)
	if err != nil {
		return "", errors.New(err.Error())
	}
	return native.JSON(b), nil
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

// Now returns the current local time.
func Now() Time {
	return NewTime(time.Now())
}

// ParseDuration parses a duration string.
// A duration string is a possibly signed sequence of
// decimal numbers, each with optional fraction and a unit suffix,
// such as "300ms", "-1.5h" or "2h45m".
// Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".
func ParseDuration(s string) (Duration, error) {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, replacePrefix(err, "time", "parseDuration")
	}
	return d, nil
}

// ParseInt interprets a string s in the given base, for 2 <= base <= 36, and
// returns the corresponding value. It returns 0 and an error if s is empty,
// contains invalid digits or the value corresponding to s cannot be
// represented by an int value.
func ParseInt(s string, base int) (int, error) {
	if base == 0 {
		return 0, errors.New("parseInt: parsing " + strconv.Quote(s) + ": invalid base 0")
	}
	i, err := strconv.ParseInt(s, base, 0)
	if err != nil {
		e := err.(*strconv.NumError)
		return 0, errors.New("parseInt: parsing " + strconv.Quote(s) + ": " + e.Err.Error())
	}
	return int(i), nil
}

// ParseFloat converts the string s to a float64 value.
//
// If s is well-formed and near a valid floating-point number, ParseFloat
// returns the nearest floating-point number rounded using IEEE754 unbiased
// rounding.
func ParseFloat(s string) (float64, error) {
	if strings.HasPrefix(s, "0x") {
		return 0, errors.New("parseFloat: parsing " + strconv.Quote(s) + ": invalid syntax")
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		e := err.(*strconv.NumError)
		return 0, errors.New("parseFloat: parsing " + strconv.Quote(s) + ": " + e.Err.Error())
	}
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, errors.New("parseFloat: parsing " + strconv.Quote(s) + ": invalid syntax")
	}
	return f, nil
}

// ParseTime parses a formatted string and returns the time value it
// represents. The layout defines the format by showing how the reference
// time would be
//	Mon Jan 2 15:04:05 -0700 MST 2006
// would be interpreted if it were the value; it serves as an example of
// the input format. The same interpretation will then be made to the
// input string.
//
// See https://golang.org/pkg/time/#Parse for more details.
//
// As a special case, if layout is an empty string, ParseTime parses a time
// representation using a predefined list of layouts.
//
// It returns an error if value cannot be parsed.
func ParseTime(layout, value string) (Time, error) {
	if layout == "" {
		for _, layout = range timeLayouts {
			t, err := time.Parse(layout, value)
			if err == nil {
				return NewTime(t), nil
			}
		}
		return Time{}, fmt.Errorf("parseTime: cannot parse \"%s\"", value)
	}
	t, err := time.Parse(layout, value)
	if err != nil {
		return Time{}, replacePrefix(err, "time", "parseTime")
	}
	return NewTime(t), nil
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

// RegExp parses a regular expression and returns a Regexp object that can be
// used to match against text. It panics if the expression cannot be parsed.
func RegExp(expr string) Regexp {
	r, err := regexp.Compile(expr)
	if err != nil {
		panic("regexp: " + err.Error())
	}
	return Regexp{r: r}
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

// Reverse returns the reverse order for data.
func Reverse(data interface{}) {
	if data == nil {
		return
	}
	rv := reflect.ValueOf(data)
	if rv.Kind() != reflect.Slice {
		panic("reverse: cannot reverse non-slice value of type " + rv.Type().String())
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
	if t := reflect.TypeOf(slice); t.Kind() != reflect.Slice {
		panic("sort: cannot sort non-slice value of type " + t.String())
	}
	if less != nil {
		if t := reflect.TypeOf(less); t.Kind() != reflect.Func {
			panic("sort: cannot sort using a non-function value of type " + t.String())
		}
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
	case []native.HTML:
		sort.Slice(s, func(i, j int) bool { return string(s[i]) < string(s[j]) })
	case []native.CSS:
		sort.Slice(s, func(i, j int) bool { return string(s[i]) < string(s[j]) })
	case []native.JS:
		sort.Slice(s, func(i, j int) bool { return string(s[i]) < string(s[j]) })
	case []native.JSON:
		sort.Slice(s, func(i, j int) bool { return string(s[i]) < string(s[j]) })
	case []native.Markdown:
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

// UnixTime returns the local Time corresponding to the given Unix time, sec
// seconds and nsec nanoseconds since January 1, 1970 UTC. It is valid to pass
// nsec outside the range [0, 999999999]. Not all sec values have a
// corresponding time value. One such value is 1<<63-1 (the largest int64
// value).
func UnixTime(sec int64, nsec int64) Time {
	return NewTime(time.Unix(sec, nsec))
}

// UnmarshalJSON parses the JSON-encoded data and stores the result in a new
// value pointed to by v. If v is nil or not a pointer, UnmarshalJSON returns
// an error.
//
// Unlike json.Unmarshal of the Go standard library, UnmarshalJSON does not
// change the value pointed to by v but instantiates a new value and then
// replaces the value pointed to by v, if no errors occur.
//
// See https://golang.org/pkg/encoding/json/#Unmarshal for details.
func UnmarshalJSON(data string, v interface{}) error {
	if v == nil {
		return errors.New("unmarshalJSON: cannot unmarshal into nil")
	}
	rv := reflect.ValueOf(v)
	rt := rv.Type()
	if rv.Kind() != reflect.Ptr {
		return fmt.Errorf("unmarshalJSON: cannot unmarshal into non-pointer value of type " + rt.String())
	}
	if rv.IsZero() {
		return errors.New("unmarshalJSON: cannot unmarshal into a nil pointer of type " + rt.String())
	}
	vp := reflect.New(rt.Elem())
	err := json.Unmarshal([]byte(data), vp.Interface())
	if err != nil {
		if e, ok := err.(*json.UnmarshalTypeError); ok {
			if e.Struct != "" || e.Field != "" {
				return errors.New("unmarshalJSON: cannot unmarshal " + e.Value + " into struct field " + e.Struct + "." + e.Field + " of type " + e.Type.String())
			}
			return errors.New("unmarshalJSON: cannot unmarshal " + e.Value + " into value of type " + e.Type.String())
		}
		return replacePrefix(err, "json", "unmarshalJSON")
	}
	rv.Elem().Set(vp.Elem())
	return nil
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

// onlyJSONWhitespace reports if s contains only JSON whitespace.
func onlyJSONWhitespace(s string) bool {
	for _, c := range s {
		switch c {
		case ' ', '\t', '\n', '\r':
		default:
			return false
		}
	}
	return true
}

// replacePrefix returns err with the prefix old replaced with new.
func replacePrefix(err error, old, new string) error {
	return errors.New(new + ": " + strings.TrimPrefix(err.Error(), old+": "))
}
