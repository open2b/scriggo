// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"math"
	"math/rand"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"scrigo"
	"scrigo/internal/compiler"
	"scrigo/vm"
)

const maxInt = int(^uint(0) >> 1)

type Hasher int

const (
	_MD5 Hasher = iota
	_SHA1
	_SHA256
)

var hasherType = reflect.TypeOf(_MD5)

var testSeed int64 = -1

var errNoSlice = errors.New("no slice")

const spaces = " \n\r\t\f" // https://infra.spec.whatwg.org/#ascii-whitespace

// TODO(Gianluca): this definition is a copy-paste from "value.go", which has
// been excluded from building. See "value.go" for further details.
type HTML string

var tcBuiltins = compiler.TypeCheckerScope{}

var builtins = scrigo.Package{
	Name: "main",
	Declarations: map[string]interface{}{
		"Hasher":      hasherType,
		"MD5":         scrigo.Constant(hasherType, _MD5),
		"SHA1":        scrigo.Constant(hasherType, _SHA1),
		"SHA256":      scrigo.Constant(hasherType, _SHA256),
		"abbreviate":  _abbreviate,
		"abs":         _abs,
		"atoi":        strconv.Atoi,
		"base64":      _base64,
		"contains":    strings.Contains,
		"errorf":      _errorf,
		"hash":        _hash,
		"hasPrefix":   strings.HasPrefix,
		"hasSuffix":   strings.HasSuffix,
		"hex":         _hex,
		"hmac":        _hmac,
		"html":        _html,
		"index":       _index,
		"indexAny":    _indexAny,
		"itoa":        strconv.Itoa,
		"join":        _join,
		"lastIndex":   _lastIndex,
		"max":         _max,
		"min":         _min,
		"rand":        _rand,
		"randFloat":   _randFloat,
		"repeat":      _repeat,
		"replace":     _replace,
		"replaceAll":  _replaceAll,
		"reverse":     _reverse,
		"round":       _round,
		"printf":      _printf,
		"shuffle":     _shuffle,
		"sort":        _sort,
		"split":       _split,
		"splitN":      _splitN,
		"queryEscape": _queryEscape,
		"title":       _title,
		"toLower":     _toLower,
		"toTitle":     _toTitle,
		"toUpper":     _toUpper,
		"trim":        strings.Trim,
		"trimLeft":    strings.TrimLeft,
		"trimPrefix":  strings.TrimPrefix,
		"trimRight":   strings.TrimRight,
		"trimSuffix":  strings.TrimSuffix,
	},
}

// _abbreviate is the builtin function "abbreviate".
func _abbreviate(env vm.Env, s string, n int) string {
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
	env.Alloc(16 + 3)
	env.Alloc(len(s))
	return s + "..."
}

// _abs is the builtin function "abs".
func _abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// _base64 is the builtin function "base64".
func _base64(env *vm.Env, s string) string {
	if s == "" {
		return s
	}
	env.Alloc(16)
	bytes := base64.StdEncoding.EncodedLen(len(s))
	if bytes < 0 {
		bytes = maxInt
	}
	env.Alloc(bytes)
	return base64.StdEncoding.EncodeToString([]byte(s))
}

// _errorf is the builtin function "errorf".
func _errorf(format string, a ...interface{}) {
	// TODO(marco): Alloc.
	panic(fmt.Errorf(format, a...))
}

// _hash is the builtin function "hash".
func _hash(env *vm.Env, hasher Hasher, s string) string {
	var h hash.Hash
	switch hasher {
	case _MD5:
		h = md5.New()
		env.Alloc(16 + 16)
	case _SHA1:
		h = sha1.New()
		env.Alloc(16 + 28)
	case _SHA256:
		h = sha256.New()
		env.Alloc(16 + 64)
	default:
		panic(errors.New("unknown hash function"))
	}
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

// _hex is the builtin function "hex".
func _hex(env *vm.Env, s string) string {
	if s == "" {
		return s
	}
	env.Alloc(16)
	bytes := hex.EncodedLen(len(s))
	if bytes < 0 {
		bytes = maxInt
	}
	env.Alloc(bytes)
	return hex.EncodeToString([]byte(s))
}

// _hmac is the builtin function "hmac".
func _hmac(env *vm.Env, hasher Hasher, message, key string) string {
	var h func() hash.Hash
	switch hasher {
	case _MD5:
		h = md5.New
		env.Alloc(16 + 24)
	case _SHA1:
		h = sha1.New
		env.Alloc(16 + 28)
	case _SHA256:
		h = sha256.New
		env.Alloc(16 + 44)
	default:
		panic(errors.New("unknown hash function"))
	}
	mac := hmac.New(h, []byte(key))
	_, _ = io.WriteString(mac, message)
	s := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return s
}

// _html is the builtin function "html".
func _html(s interface{}) HTML {
	switch v := s.(type) {
	case string:
		return HTML(v)
	case HTML:
		return v
	}
	panic(fmt.Sprintf("type-checking bug: _html argument must be string or HTML, got %T", s))
}

// _index is the builtin function "index".
func _index(s, substr string) int {
	n := strings.Index(s, substr)
	if n <= 1 {
		return n
	}
	return utf8.RuneCountInString(s[0:n])
}

// _indexAny is the builtin function "indexAny".
func _indexAny(s, chars string) int {
	n := strings.IndexAny(s, chars)
	if n <= 1 {
		return n
	}
	return utf8.RuneCountInString(s[0:n])
}

// _join is the builtin function "join".
func _join(env *vm.Env, a []string, sep string) string {
	if n := len(a); n > 1 {
		env.Alloc(16)
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

// _lastIndex is the builtin function "lastIndex".
func _lastIndex(s, sep string) int {
	n := strings.LastIndex(s, sep)
	if n <= 1 {
		return n
	}
	return utf8.RuneCountInString(s[0:n])
}

// _max is the builtin function "max".
func _max(x, y int) int {
	if x < y {
		return y
	}
	return x
}

// _min is the builtin function "min".
func _min(x, y int) int {
	if y < x {
		return y
	}
	return x
}

// _printf is the builtin function "printf".
func _printf(format string, a ...interface{}) (n int, err error) {
	// TODO(marco): Alloc.
	return fmt.Printf(format, a...)
}

// _rand is the builtin function "rand".
func _rand(x int) int {
	seed := time.Now().UTC().UnixNano()
	if testSeed >= 0 {
		seed = testSeed
	}
	r := rand.New(rand.NewSource(seed))
	switch {
	case x < 0:
		return r.Int()
	case x == 0:
		panic("invalid argument to rand")
	default:
		return r.Intn(x)
	}
}

// _randFloat is the builtin function "randFloat".
func _randFloat() float64 {
	return rand.Float64()
}

// _repeat is the builtin function "repeat".
func _repeat(env *vm.Env, s string, count int) string {
	if count < 0 {
		panic("negative repeat count")
	}
	if s == "" || count == 0 {
		return s
	}
	env.Alloc(16)
	bytes := len(s) * count
	if bytes/count != len(s) {
		bytes = maxInt
	}
	env.Alloc(bytes)
	return strings.Repeat(s, count)
}

// _replace is the builtin function "replace".
func _replace(env *vm.Env, s, old, new string, n int) string {
	if old != new && n != 0 {
		if c := strings.Count(s, old); c > 0 {
			if n > 0 && c > n {
				c = n
			}
			env.Alloc(16)
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

// _replaceAll is the builtin function "replaceAll".
func _replaceAll(env *vm.Env, s, old, new string) string {
	return _replace(env, s, old, new, -1)
}

// _reverse is the builtin function "reverse".
func _reverse(s interface{}) interface{} {
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

// _round is the builtin function "round".
func _round(x float64) float64 {
	return math.Round(x)
}

// _shuffle is the builtin function "shuffle".
func _shuffle(s interface{}) []interface{} {
	if s == nil {
		return nil
	}
	var ms []interface{}
	switch m := s.(type) {
	case []interface{}:
		ms = make([]interface{}, len(m))
		copy(ms, m)
	default:
		rv := reflect.ValueOf(s)
		if rv.Kind() != reflect.Slice {
			panic(errNoSlice)
		}
		l := rv.Len()
		ms = make([]interface{}, l)
		for i := 0; i < l; i++ {
			ms[i] = rv.Index(i).Interface()
		}
	}
	if len(ms) < 2 {
		return ms
	}
	// Swap.
	seed := time.Now().UTC().UnixNano()
	if testSeed >= 0 {
		seed = testSeed
	}
	r := rand.New(rand.NewSource(seed))
	swap := reflect.Swapper(ms)
	for i := len(ms) - 1; i >= 0; i-- {
		j := r.Intn(i + 1)
		swap(i, j)
	}
	return ms
}

// _sort is the builtin function "sort".
func _sort(slice interface{}) {
	// no reflect
	switch s := slice.(type) {
	case nil:
	case []string:
		sort.Strings(s)
	case []HTML:
		sort.Slice(s, func(i, j int) bool { return string(s[i]) < string(s[j]) })
	case []rune:
		sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	case []byte:
		sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	case []int:
		sort.Ints(s)
	case []float64:
		sort.Float64s(s)
	}
	// reflect
	sortSlice(slice)
}

// _split is the builtin function "split".
func _split(env *vm.Env, s, sep string) []string {
	return _splitN(env, s, sep, -1)
}

// _splitN is the builtin function "splitN".
func _splitN(env *vm.Env, s, sep string, n int) []string {
	if n != 0 {
		env.Alloc(16)
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

// _queryEscape is the builtin function "queryEscape".
func _queryEscape(env *vm.Env, s string) string {
	alloc := false
	numHex := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if '0' <= c && c <= '9' || 'A' <= c && c <= 'Z' || 'a' <= c && c <= 'z' ||
			c == '-' || c == '.' || c == '_' || c == '~' {
			continue
		}
		alloc = true
		if c != ' ' {
			numHex++
		}
	}
	if alloc {
		env.Alloc(16)
		env.Alloc(len(s))
		if 2*numHex < 0 {
			env.Alloc(maxInt)
		}
		env.Alloc(2 * numHex)
	}
	return url.QueryEscape(s)
}

// _title is the builtin function "title".
func _title(env *vm.Env, s string) string {
	return withAlloc(env, strings.Title, s)
}

// _toLower is the builtin function "toLower".
func _toLower(env *vm.Env, s string) string {
	return withAlloc(env, strings.ToLower, s)
}

// _toTitle is the builtin function "toTitle".
func _toTitle(env *vm.Env, s string) string {
	return withAlloc(env, strings.ToTitle, s)
}

// _toUpper is the builtin function "toUpper".
func _toUpper(env *vm.Env, s string) string {
	return withAlloc(env, strings.ToUpper, s)
}

// withAlloc wraps a function that get a string and return a string,
// allocating memory only if necessary. Should be used only if the function f
// guarantees that if the output string is a new allocated string it is not
// equal to the input string.
func withAlloc(env *vm.Env, f func(string) string, s string) string {
	env.Alloc(16)
	env.Alloc(len(s))
	t := f(s)
	if t == s {
		env.Alloc(-16)
		env.Alloc(-len(s))
	} else {
		env.Alloc(len(t) - len(s))
	}
	return t
}
