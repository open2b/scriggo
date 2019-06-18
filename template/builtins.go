// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

import (
	_hmac "crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	_base64 "encoding/base64"
	_hex "encoding/hex"
	"errors"
	"fmt"
	_hash "hash"
	_html "html"
	"io"
	"math"
	_rand "math/rand"
	"reflect"
	_sort "sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"scriggo"
	"scriggo/vm"
)

const maxInt = int(^uint(0) >> 1)

type hasher int

const (
	_MD5 hasher = iota
	_SHA1
	_SHA256
)

var hasherType = reflect.TypeOf(_MD5)
var timeType = reflect.TypeOf(Time{})

var testSeed int64 = -1

var errNoSlice = errors.New("no slice")

const spaces = " \n\r\t\f" // https://infra.spec.whatwg.org/#ascii-whitespace

func Builtins() *scriggo.Package {
	p := scriggo.Package{
		Name:         "main",
		Declarations: map[string]interface{}{},
	}
	for name, value := range main.Declarations {
		p.Declarations[name] = value
	}
	return &p
}

type HTML string

func (h HTML) Render(out io.Writer, ctx Context) error {
	w := newStringWriter(out)
	var err error
	switch ctx {
	case ContextText:
		_, err = w.WriteString(string(h))
	case ContextHTML:
		_, err = w.WriteString(string(h))
	case ContextTag:
		err = renderInTag(w, string(h))
	case ContextAttribute:
		//if urlstate == nil {
		err = attributeEscape(w, _html.UnescapeString(string(h)), true)
		//} else {
		//	err = r.renderInAttributeURL(w, value, node, urlstate, true)
		//}
	case ContextUnquotedAttribute:
		//if urlstate == nil {
		err = attributeEscape(w, _html.UnescapeString(string(h)), false)
		//} else {
		//	err = r.renderInAttributeURL(w, value, node, urlstate, false)
		//}
	case ContextCSS:
		_, err := w.WriteString(`"`)
		if err != nil {
			return err
		}
		err = cssStringEscape(w, string(h))
		if err != nil {
			return err
		}
		_, err = w.WriteString(`"`)
	case ContextCSSString:
		err = cssStringEscape(w, string(h))
	case ContextJavaScript:
		_, err := w.WriteString("\"")
		if err != nil {
			return err
		}
		err = javaScriptStringEscape(w, string(h))
		if err != nil {
			return err
		}
		_, err = w.WriteString("\"")
	case ContextJavaScriptString:
		err = javaScriptStringEscape(w, string(h))
	default:
		panic("scriggo: unknown context")
	}
	return err
}

var times sync.Map

type Time time.Time

func (t Time) Format(layout string) string {
	return time.Time(t).Format(layout)
}

func (t Time) String() string {
	return time.Time(t).String()
}

func (t Time) UTC(env *vm.Env) Time {
	env.Alloc(24)
	return Time(time.Time(t).UTC())
}

func (t Time) Render(out io.Writer, ctx Context) error {
	var err error
	switch ctx {
	case ContextText:
		w := newStringWriter(out)
		_, err = w.WriteString(t.Format(time.RFC3339))
	case ContextHTML:
		w := newStringWriter(out)
		_, err = w.WriteString(t.Format(time.RFC1123))
	case ContextAttribute, ContextUnquotedAttribute:
		w := newStringWriter(out)
		//if urlstate == nil {
		_, err = w.WriteString(t.Format(time.RFC3339))
		//} else {
		//	err = r.renderInAttributeURL(w, value, node, urlstate, true)
		//}
	case ContextJavaScript:
		tt := time.Time(t)
		year := tt.Year()
		if year < 100 {
			return errors.New("not representable year in JavaScript")
		}
		b := make([]byte, 0, 50)
		b = append(b, "new Date("...)
		b = strconv.AppendInt(b, int64(year), 10)
		b = append(b, ',', ' ')
		b = strconv.AppendInt(b, int64(tt.Month()), 10)
		b = append(b, ',', ' ')
		b = strconv.AppendInt(b, int64(tt.Day()), 10)
		b = append(b, ',', ' ')
		b = strconv.AppendInt(b, int64(tt.Hour()), 10)
		b = append(b, ',', ' ')
		b = strconv.AppendInt(b, int64(tt.Minute()), 10)
		b = append(b, ',', ' ')
		b = strconv.AppendInt(b, int64(tt.Second()), 10)
		b = append(b, ',', ' ')
		b = strconv.AppendInt(b, int64(tt.Nanosecond())/int64(time.Millisecond), 10)
		b = append(b, ')')
		_, err = out.Write(b)
	case ContextJavaScriptString:
		tt := time.Time(t)
		year := tt.Year()
		if year < 0 || year > 9999 {
			return errors.New("not representable year in JavaScript")
		}
		w := newStringWriter(out)
		_, err = w.WriteString(tt.Format("2006-01-02T15:04:05.000Z07:00"))
	default:
		err = ErrNoRenderInContext
	}
	return err
}

var main = scriggo.Package{
	Name: "main",
	Declarations: map[string]interface{}{
		"Hasher":      hasherType,
		"MD5":         scriggo.ConstantValue(_MD5),
		"SHA1":        scriggo.ConstantValue(_SHA1),
		"SHA256":      scriggo.ConstantValue(_SHA256),
		"Time":        timeType,
		"abbreviate":  abbreviate,
		"abs":         abs,
		"atoi":        strconv.Atoi,
		"base64":      base64,
		"contains":    strings.Contains,
		"errorf":      errorf,
		"escapeHTML":  escapeHTML,
		"escapeQuery": escapeQuery,
		"hash":        hash,
		"hasPrefix":   strings.HasPrefix,
		"hasSuffix":   strings.HasSuffix,
		"hex":         hex,
		"hmac":        hmac,
		"html":        reflect.TypeOf(HTML("")),
		"index":       index,
		"indexAny":    indexAny,
		"itoa":        itoa,
		"join":        join,
		"lastIndex":   lastIndex,
		"max":         max,
		"min":         min,
		"now":         now,
		"rand":        rand,
		"randFloat":   randFloat,
		"repeat":      repeat,
		"replace":     replace,
		"replaceAll":  replaceAll,
		"reverse":     reverse,
		"round":       round,
		"printf":      printf,
		"shuffle":     shuffle,
		"sort":        sort,
		"split":       split,
		"splitN":      splitN,
		"title":       title,
		"toLower":     toLower,
		"toTitle":     toTitle,
		"toUpper":     toUpper,
		"trim":        strings.Trim,
		"trimLeft":    strings.TrimLeft,
		"trimPrefix":  strings.TrimPrefix,
		"trimRight":   strings.TrimRight,
		"trimSuffix":  strings.TrimSuffix,
	},
}

// abbreviate is the builtin function "abbreviate".
func abbreviate(env *vm.Env, s string, n int) string {
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
	env.Alloc(3)
	env.Alloc(len(s))
	return s + "..."
}

// abs is the builtin function "abs".
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// base64 is the builtin function "base64".
func base64(env *vm.Env, s string) string {
	if s == "" {
		return s
	}
	bytes := _base64.StdEncoding.EncodedLen(len(s))
	if bytes < 0 {
		bytes = maxInt
	}
	env.Alloc(bytes)
	return _base64.StdEncoding.EncodeToString([]byte(s))
}

// errorf is the builtin function "errorf".
func errorf(format string, a ...interface{}) {
	// TODO(marco): Alloc.
	panic(fmt.Errorf(format, a...))
}

// escapeHTML is the builtin function "escapeHTML".
func escapeHTML(env *vm.Env, s string) HTML {
	more := 0
	for i := 0; i < len(s); i++ {
		switch c := s[i]; c {
		case '<', '>':
			more += 3
		case '&', '\'', '"':
			more += 4
		}
	}
	if more == 0 {
		return HTML(s)
	}
	env.Alloc(len(s) + more)
	b := make([]byte, len(s)+more)
	for i, j := 0, 0; i < len(s); i++ {
		switch c := s[i]; c {
		case '<', '>':
			b[j] = '&'
			if c == '<' {
				b[j+1] = 'l'
			} else {
				b[j+1] = 'g'
			}
			b[j+2] = 't'
			b[j+3] = ';'
			j += 4
		case '&':
			b[j] = '&'
			b[j+1] = 'a'
			b[j+2] = 'm'
			b[j+3] = 'p'
			b[j+4] = ';'
			j += 5
		case '"', '\'':
			b[j] = '&'
			b[j+1] = '#'
			b[j+2] = '3'
			if c == '"' {
				b[j+3] = '4'
			} else {
				b[j+3] = '9'
			}
			b[j+4] = ';'
			j += 5
		default:
			b[j] = c
			j++
		}
	}
	return HTML(b)
}

// escapeQuery is the builtin function "escapeQuery".
func escapeQuery(env *vm.Env, s string) string {
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
	// Alloc memory.
	env.Alloc(len(s))
	if 2*numHex < 0 {
		env.Alloc(maxInt)
	}
	env.Alloc(2 * numHex)
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

// hash is the builtin function "hash".
func hash(env *vm.Env, hasher hasher, s string) string {
	var h _hash.Hash
	switch hasher {
	case _MD5:
		h = md5.New()
		env.Alloc(16)
	case _SHA1:
		h = sha1.New()
		env.Alloc(28)
	case _SHA256:
		h = sha256.New()
		env.Alloc(64)
	default:
		panic("call of hash with unknown hasher")
	}
	h.Write([]byte(s))
	return _hex.EncodeToString(h.Sum(nil))
}

// hex is the builtin function "hex".
func hex(env *vm.Env, s string) string {
	if s == "" {
		return s
	}
	bytes := _hex.EncodedLen(len(s))
	if bytes < 0 {
		bytes = maxInt
	}
	env.Alloc(bytes)
	return _hex.EncodeToString([]byte(s))
}

// hmac is the builtin function "hmac".
func hmac(env *vm.Env, hasher hasher, message, key string) string {
	var h func() _hash.Hash
	switch hasher {
	case _MD5:
		h = md5.New
		env.Alloc(24)
	case _SHA1:
		h = sha1.New
		env.Alloc(28)
	case _SHA256:
		h = sha256.New
		env.Alloc(44)
	default:
		panic("call of hmac with unknown hasher")
	}
	mac := _hmac.New(h, []byte(key))
	_, _ = io.WriteString(mac, message)
	s := _base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return s
}

// index is the builtin function "index".
func index(s, substr string) int {
	n := strings.Index(s, substr)
	if n <= 1 {
		return n
	}
	return utf8.RuneCountInString(s[0:n])
}

// indexAny is the builtin function "indexAny".
func indexAny(s, chars string) int {
	n := strings.IndexAny(s, chars)
	if n <= 1 {
		return n
	}
	return utf8.RuneCountInString(s[0:n])
}

// itoa is the builtin function "itoa".
func itoa(env *vm.Env, i int) string {
	s := strconv.Itoa(i)
	env.Alloc(len(s))
	return s
}

// join is the builtin function "join".
func join(env *vm.Env, a []string, sep string) string {
	if n := len(a); n > 1 {
		bytes := (n - 1) * len(sep)
		if bytes/(n-1) != len(sep) {
			bytes = maxInt
		}
		env.Alloc(bytes)
		for _, s := range a {
			env.Alloc(len(s))
		}
	}
	return strings.Join(a, sep)
}

// lastIndex is the builtin function "lastIndex".
func lastIndex(s, sep string) int {
	n := strings.LastIndex(s, sep)
	if n <= 1 {
		return n
	}
	return utf8.RuneCountInString(s[0:n])
}

// max is the builtin function "max".
func max(x, y int) int {
	if x < y {
		return y
	}
	return x
}

// min is the builtin function "min".
func min(x, y int) int {
	if y < x {
		return y
	}
	return x
}

// now is the builtin function "now".
func now(env *vm.Env) Time {
	if env.Context() == nil {
		return Time(time.Now())
	}
	t := Time(time.Now().Round(1 * time.Millisecond))
	actual, loaded := times.LoadOrStore(env, t)
	if loaded {
		env.ExitFunc(func() {
			times.Delete(env)
		})
	}
	return actual.(Time)
}

// printf is the builtin function "printf".
func printf(format string, a ...interface{}) (n int, err error) {
	// TODO(marco): Alloc.
	return fmt.Printf(format, a...)
}

// rand is the builtin function "rand".
func rand(x int) int {
	seed := time.Now().UTC().UnixNano()
	if testSeed >= 0 {
		seed = testSeed
	}
	r := _rand.New(_rand.NewSource(seed))
	switch {
	case x < 0:
		return r.Int()
	case x == 0:
		panic("call of rand with zero value")
	default:
		return r.Intn(x)
	}
}

// randFloat is the builtin function "randFloat".
func randFloat() float64 {
	return _rand.Float64()
}

// repeat is the builtin function "repeat".
func repeat(env *vm.Env, s string, count int) string {
	if count < 0 {
		panic("call of repeat with negative count")
	}
	if s == "" || count == 0 {
		return ""
	}
	bytes := len(s) * count
	if bytes/count != len(s) {
		bytes = maxInt
	}
	env.Alloc(bytes)
	return strings.Repeat(s, count)
}

// replace is the builtin function "replace".
func replace(env *vm.Env, s, old, new string, n int) string {
	if old != new && n != 0 {
		if c := strings.Count(s, old); c > 0 {
			if n > 0 && c > n {
				c = n
			}
			env.Alloc(len(s))
			bytes := (len(new) - len(old)) * c
			if bytes/c != len(new)-len(old) {
				bytes = maxInt
			}
			env.Alloc(bytes)
		}
	}
	return strings.Replace(s, old, new, n)
}

// replaceAll is the builtin function "replaceAll".
func replaceAll(env *vm.Env, s, old, new string) string {
	return replace(env, s, old, new, -1)
}

// reverse is the builtin function "reverse".
func reverse(s interface{}) interface{} {
	if s == nil {
		return nil
	}
	rv := reflect.ValueOf(s)
	if rv.Kind() != reflect.Slice {
		panic(errNoSlice)
	}
	l := rv.Len()
	if l <= 1 {
		return s
	}
	rt := reflect.TypeOf(s)
	rvc := reflect.MakeSlice(rt, l, l)
	reflect.Copy(rvc, rv)
	sc := rvc.Interface()
	swap := reflect.Swapper(sc)
	for i, j := 0, l-1; i < j; i, j = i+1, j-1 {
		swap(i, j)
	}
	return sc
}

// round is the builtin function "round".
func round(x float64) float64 {
	return math.Round(x)
}

// shuffle is the builtin function "shuffle".
func shuffle(env *vm.Env, slice interface{}) {
	if slice == nil {
		return
	}
	v := reflect.ValueOf(slice)
	if v.Kind() != reflect.Slice {
		err := fmt.Sprintf("call of shuffle on %s value", v.Kind())
		env.Alloc(len(err))
		panic(err)
	}
	// Swap.
	seed := time.Now().UTC().UnixNano()
	if testSeed >= 0 {
		seed = testSeed
	}
	r := _rand.New(_rand.NewSource(seed))
	swap := reflect.Swapper(slice)
	for i := v.Len() - 1; i >= 0; i-- {
		j := r.Intn(i + 1)
		swap(i, j)
	}
	return
}

// sort is the builtin function "sort".
func sort(slice interface{}) {
	// no reflect
	switch s := slice.(type) {
	case nil:
	case []string:
		_sort.Strings(s)
	case []rune:
		_sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	case []byte:
		_sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	case []HTML:
		_sort.Slice(s, func(i, j int) bool { return string(s[i]) < string(s[j]) })
	case []int:
		_sort.Ints(s)
	case []float64:
		_sort.Float64s(s)
	}
	// reflect
	sortSlice(slice)
}

// split is the builtin function "split".
func split(env *vm.Env, s, sep string) []string {
	return splitN(env, s, sep, -1)
}

// splitN is the builtin function "splitN".
func splitN(env *vm.Env, s, sep string, n int) []string {
	if n != 0 {
		if sep == "" {
			// Invalid UTF-8 sequences are replaced with RuneError.
			i := 0
			for _, r := range s {
				if i == n {
					break
				}
				env.Alloc(16 + utf8.RuneLen(r)) // string head and bytes.
				i++
			}
		} else {
			c := strings.Count(s, sep)
			if n > 0 && c > n {
				c = n
			}
			bytes := 16 * (c + 1) // string heads.
			if bytes < 0 {
				bytes = maxInt
			}
			env.Alloc(bytes)
			env.Alloc(len(s) - len(sep)*c) // string bytes.
		}
	}
	return strings.SplitN(s, sep, n)
}

// title is the builtin function "title".
func title(env *vm.Env, s string) string {
	return withAlloc(env, strings.Title, s)
}

// toLower is the builtin function "toLower".
func toLower(env *vm.Env, s string) string {
	return withAlloc(env, strings.ToLower, s)
}

// toTitle is the builtin function "toTitle".
func toTitle(env *vm.Env, s string) string {
	return withAlloc(env, strings.ToTitle, s)
}

// toUpper is the builtin function "toUpper".
func toUpper(env *vm.Env, s string) string {
	return withAlloc(env, strings.ToUpper, s)
}

// withAlloc wraps a function that get a string and return a string,
// allocating memory only if necessary. Should be used only if the function f
// guarantees that if the output string is a new allocated string it is not
// equal to the input string.
func withAlloc(env *vm.Env, f func(string) string, s string) string {
	env.Alloc(len(s))
	t := f(s)
	if t == s {
		env.Alloc(-len(s))
	} else {
		env.Alloc(len(t) - len(s))
	}
	return t
}
