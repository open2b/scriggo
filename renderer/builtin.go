//
// Copyright (c) 2016-2018 Open2b Software Snc. All Rights Reserved.
//

package renderer

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
	"math/rand"
	"net/url"
	"reflect"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/shopspring/decimal"
)

var testSeed int64 = -1

var errNoSlice = errors.New("no slice")

const spaces = " \n\r\t\f" // https://infra.spec.whatwg.org/#ascii-whitespace

var builtins = map[string]interface{}{
	"nil":    nil,
	"true":   true,
	"false":  false,
	"len":    _len,
	"string": _string,
	"number": _number,

	"abbreviate":  _abbreviate,
	"abs":         _abs,
	"contains":    _contains,
	"errorf":      _errorf,
	"hasPrefix":   _hasPrefix,
	"hasSuffix":   _hasSuffix,
	"hmac":        _hmac,
	"html":        _html,
	"index":       _index,
	"indexAny":    _indexAny,
	"int":         _int,
	"join":        _join,
	"lastIndex":   _lastIndex,
	"max":         _max,
	"md5":         _md5,
	"min":         _min,
	"rand":        _rand,
	"repeat":      _repeat,
	"replace":     _replace,
	"reverse":     _reverse,
	"round":       _round,
	"sha1":        _sha1,
	"sha256":      _sha256,
	"shuffle":     _shuffle,
	"sort":        _sort,
	"split":       _split,
	"splitAfter":  _splitAfter,
	"queryEscape": _queryEscape,
	"title":       _title,
	"toLower":     _toLower,
	"toTitle":     _toTitle,
	"toUpper":     _toUpper,
	"trim":        _trim,
	"trimLeft":    _trimLeft,
	"trimPrefix":  _trimPrefix,
	"trimRight":   _trimRight,
	"trimSuffix":  _trimSuffix,
}

// _abbreviate is the builtin function "abbreviate".
func _abbreviate(s string, n int) string {
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

// _abs is the builtin function "abs".
func _abs(d decimal.Decimal) decimal.Decimal {
	if d.Cmp(zero) < 0 {
		return d.Neg()
	}
	return d
}

// _contains is the builtin function "contains".
func _contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// _errorf is the builtin function "errorf".
func _errorf(format string, a ...interface{}) {
	panic(fmt.Errorf(format, a...))
}

// _hasPrefix is the builtin function "hasPrefix".
func _hasPrefix(s, prefix string) bool {
	return strings.HasPrefix(s, prefix)
}

// _hasSuffix is the builtin function "hasSuffix".
func _hasSuffix(s, suffix string) bool {
	return strings.HasSuffix(s, suffix)
}

// _hmac is the builtin function "hmac".
func _hmac(hasher, message, key string) string {
	var h func() hash.Hash
	switch hasher {
	case "MD5":
		h = md5.New
	case "SHA-1":
		h = sha1.New
	case "SHA-256":
		h = sha256.New
	default:
		panic("unknown hash function")
	}
	mac := hmac.New(h, []byte(key))
	io.WriteString(mac, message)
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// _html is the builtin function "html".
func _html(s string) HTML {
	return HTML(s)
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

// _int is the builtin function "int".
func _int(d decimal.Decimal) decimal.Decimal {
	return d.Truncate(0)
}

// _join is the builtin function "join".
func _join(a []string, sep string) string {
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

// _len is the builtin function "len".
func _len(v interface{}) int {
	if v == nil {
		// TODO(marco): returns a different error for "len(nil)"
		panic(fmt.Sprintf("missing argument to len: len()"))
	}
	switch s := v.(type) {
	case string:
		if len(s) <= 1 {
			return len(s)
		}
		return utf8.RuneCountInString(s)
	case HTML:
		if len(string(s)) <= 1 {
			return len(string(s))
		}
		return utf8.RuneCountInString(string(s))
		return 0
	case []int:
		return len(s)
	case []decimal.Decimal:
		return len(s)
	case []string:
		return len(s)
	case []HTML:
		return len(s)
	case []bool:
		return len(s)
	case []interface{}:
		return len(s)
	default:
		var rv = reflect.ValueOf(v)
		if rv.Kind() == reflect.Slice {
			return rv.Len()
		}
	}
	// TODO(marco): report the argument in the error as in Go
	panic(fmt.Sprintf("invalid argument (type %s) for len", typeof(v)))
}

// _max is the builtin function "max".
func _max(a, b decimal.Decimal) decimal.Decimal {
	if a.Cmp(b) < 0 {
		return b
	}
	return a
}

// _md5 is the builtin function "md5".
func _md5(s string) string {
	hasher := md5.New()
	hasher.Write([]byte(s))
	return hex.EncodeToString(hasher.Sum(nil))
}

// _min is the builtin function "min".
func _min(a, b decimal.Decimal) decimal.Decimal {
	if a.Cmp(b) > 0 {
		return b
	}
	return a
}

// _number is the builtin function "number".
func _number(d decimal.Decimal) decimal.Decimal {
	return d
}

// _queryEscape is the builtin function "queryEscape".
func _queryEscape(s string) string {
	return url.QueryEscape(s)
}

// _rand is the builtin function "rand".
func _rand(d int) decimal.Decimal {
	// seed
	seed := time.Now().UTC().UnixNano()
	if testSeed >= 0 {
		seed = testSeed
	}
	r := rand.New(rand.NewSource(seed))
	var rn int
	if d > 0 {
		rn = r.Intn(d)
	} else {
		rn = r.Int()
	}
	return decimal.New(int64(rn), 0)
}

// _repeat is the builtin function "repeat".
func _repeat(s string, count int) string {
	return strings.Repeat(s, count)
}

// _replace is the builtin function "replace".
func _replace(s, old, new string) string {
	return strings.Replace(s, old, new, -1)
}

// _reverse is the builtin function "reverse".
func _reverse(s interface{}) interface{} {
	if s == nil {
		return s
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
func _round(d decimal.Decimal, places int) decimal.Decimal {
	return d.Round(int32(places))
}

// _sha1 is the builtin function "sha1".
func _sha1(s string) string {
	hasher := sha1.New()
	hasher.Write([]byte(s))
	return hex.EncodeToString(hasher.Sum(nil))
}

// _sha256 is the builtin function "sha256".
func _sha256(s string) string {
	hasher := sha256.New()
	hasher.Write([]byte(s))
	return hex.EncodeToString(hasher.Sum(nil))
}

// _shuffle is the builtin function "shuffle".
func _shuffle(s interface{}) interface{} {
	if s == nil {
		return nil
	}
	rv := reflect.ValueOf(s)
	if rv.Kind() != reflect.Slice {
		panic(errNoSlice)
	}
	l := rv.Len()
	if l < 2 {
		return s
	}
	// Seed.
	seed := time.Now().UTC().UnixNano()
	if testSeed >= 0 {
		seed = testSeed
	}
	r := rand.New(rand.NewSource(seed))
	// Swap.
	rv2 := reflect.MakeSlice(rv.Type(), l, l)
	reflect.Copy(rv2, rv)
	s2 := rv2.Interface()
	swap := reflect.Swapper(s2)
	for i := l - 1; i >= 0; i-- {
		j := r.Intn(i + 1)
		swap(i, j)
	}
	return s2
}

// _sort is the builtin function "sort".
func _sort(slice interface{}, field string) interface{} {
	if field != "" {
		r, _ := utf8.DecodeRuneInString(field)
		if r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			panic("invalid field")
		}
	}
	if slice == nil {
		return slice
	}
	// no reflect
	switch s := slice.(type) {
	case []string:
		if len(s) <= 1 {
			return s
		}
		s2 := make([]string, len(s))
		copy(s2, s)
		sort.Strings(s2)
		return s2
	case []int:
		if len(s) <= 1 {
			return s
		}
		s2 := make([]int, len(s))
		copy(s2, s)
		sort.Ints(s2)
		return s2
	case []bool:
		if len(s) <= 1 {
			return s
		}
		s2 := make([]bool, len(s))
		copy(s2, s)
		sort.Slice(s2, func(i, j int) bool { return !s2[i] })
		return s2
	}
	// reflect
	if field == "" {
		panic("missing field")
	}
	defer func() {
		if r := recover(); r != nil {
			panic("unsortable")
		}
	}()
	rv := reflect.ValueOf(slice)
	size := rv.Len()
	values := make([]interface{}, size)
	for i := 0; i < size; i++ {
		v := rv.Index(i)
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
		values[i] = v.FieldByName(field).Interface()
	}
	var value interface{}
	if len(values) > 0 {
		value = values[0]
	} else {
		value = reflect.Zero(rv.Type()).Interface()
	}
	var f func(int, int) bool
	switch value.(type) {
	case Stringer:
		if size <= 1 {
			return slice
		}
		vv := make([]string, size)
		for i := 0; i < size; i++ {
			vv[i] = values[i].(Stringer).String()
		}
		f = func(i, j int) bool { return vv[i] < vv[j] }
	case Numberer:
		if size <= 1 {
			return slice
		}
		vv := make([]decimal.Decimal, size)
		for i := 0; i < size; i++ {
			vv[i] = values[i].(Numberer).Number()
		}
		f = func(i, j int) bool { return vv[i].Cmp(vv[j]) < 0 }
	case string:
		if size <= 1 {
			return slice
		}
		f = func(i, j int) bool { return values[i].(string) < values[j].(string) }
	case int:
		if size <= 1 {
			return slice
		}
		f = func(i, j int) bool { return values[i].(int) < values[j].(int) }
	case bool:
		if size <= 1 {
			return slice
		}
		f = func(i, j int) bool { return !values[i].(bool) }
	}
	rv2 := reflect.MakeSlice(rv.Type(), size, size)
	reflect.Copy(rv2, rv)
	s2 := rv2.Interface()
	sort.Slice(s2, f)
	return s2
}

// _split is the builtin function "split".
func _split(s, sep string, n interface{}) []string {
	n2 := -1
	if n != nil {
		var ok bool
		n2, ok = n.(int)
		if !ok {
			panic("not int parameter")
		}
	}
	return strings.SplitN(s, sep, n2)
}

// _splitAfter is the builtin function "splitAfter".
func _splitAfter(s, sep string, n interface{}) []string {
	n2 := -1
	if n != nil {
		var ok bool
		n2, ok = n.(int)
		if !ok {
			panic("not int parameter")
		}
	}
	return strings.SplitAfterN(s, sep, n2)
}

// _string is the builtin function "string".
func _string(s string) string {
	return s
}

// _title is the builtin function "title".
func _title(s string) string {
	return strings.Title(s)
}

// _toLower is the builtin function "toLower".
func _toLower(s string) string {
	return strings.ToLower(s)
}

// _toTitle is the builtin function "toTitle".
func _toTitle(s string) string {
	return strings.ToTitle(s)
}

// _toUpper is the builtin function "toUpper".
func _toUpper(s string) string {
	return strings.ToUpper(s)
}

// _trim is the builtin function "trim".
func _trim(s string, cutset interface{}) string {
	if cutset == nil {
		return strings.TrimSpace(s)
	}
	if cut, ok := cutset.(string); ok {
		return strings.Trim(s, cut)
	} else {
		panic("not string parameter")
	}
}

// _trimLeft is the builtin function "trimLeft".
func _trimLeft(s string, cutset interface{}) string {
	if cutset == nil {
		return strings.TrimLeft(s, spaces)
	}
	if cut, ok := cutset.(string); ok {
		return strings.TrimLeft(s, cut)
	} else {
		panic("not string parameter")
	}
}

// _trimRight is the builtin function "trimRight".
func _trimRight(s string, cutset interface{}) string {
	if cutset == nil {
		return strings.TrimRight(s, spaces)
	}
	if cut, ok := cutset.(string); ok {
		return strings.TrimRight(s, cut)
	} else {
		panic("not string parameter")
	}
}

// _trimPrefix is the builtin function "trimPrefix".
func _trimPrefix(s, prefix string) string {
	return strings.TrimPrefix(s, prefix)
}

// _trimSuffix is the builtin function "trimSuffix".
func _trimSuffix(s, suffix string) string {
	return strings.TrimPrefix(s, suffix)
}
