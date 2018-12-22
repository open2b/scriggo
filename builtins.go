// Copyright (c) 2018 Open2b Software Snc. All rights reserved.
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

type valuetype string

var builtins = map[string]interface{}{
	"nil":    nil,
	"true":   true,
	"false":  false,
	"len":    _len,
	"append": _append,
	"delete": _delete,

	"string": valuetype("string"),
	"html":   valuetype("html"),
	"number": valuetype("number"),
	"int":    valuetype("int"),
	"bool":   valuetype("bool"),
	"map":    valuetype("map"),
	"slice":  valuetype("slice"),
	"error":  valuetype("error"),

	"abbreviate":  _abbreviate,
	"abs":         _abs,
	"base64":      _base64,
	"contains":    strings.Contains,
	"errorf":      _errorf,
	"hasPrefix":   strings.HasPrefix,
	"hasSuffix":   strings.HasSuffix,
	"hex":         _hex,
	"hmac":        _hmac,
	"index":       _index,
	"indexAny":    _indexAny,
	"join":        strings.Join,
	"lastIndex":   _lastIndex,
	"max":         _max,
	"md5":         _md5,
	"min":         _min,
	"rand":        _rand,
	"repeat":      _repeat,
	"replace":     strings.Replace,
	"replaceAll":  _replaceAll,
	"reverse":     _reverse,
	"round":       _round,
	"sha1":        _sha1,
	"sha256":      _sha256,
	"shuffle":     _shuffle,
	"sort":        _sort,
	"sortBy":      _sortBy,
	"split":       strings.Split,
	"splitN":      strings.SplitN,
	"queryEscape": url.QueryEscape,
	"title":       strings.Title,
	"toLower":     strings.ToLower,
	"toTitle":     strings.ToTitle,
	"toUpper":     strings.ToUpper,
	"trim":        strings.Trim,
	"trimLeft":    strings.TrimLeft,
	"trimPrefix":  strings.TrimPrefix,
	"trimRight":   strings.TrimRight,
	"trimSuffix":  strings.TrimSuffix,
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

// _append is the builtin function "append".
func _append(slice interface{}, elems ...interface{}) MutableSlice {
	if slice == nil {
		panic(fmt.Errorf("first argument to append must be slice; have nil"))
	}
	switch s := slice.(type) {
	case MutableSlice:
		return append(s, elems...)
	case []interface{}:
		l := len(s)
		ms := make(MutableSlice, l+len(elems))
		for i, v := range s {
			ms[i] = v
		}
		for i, v := range elems {
			ms[l+i] = v
		}
		return ms
	case []string:
		l := len(s)
		ms := make(MutableSlice, l+len(elems))
		for i, v := range s {
			ms[i] = v
		}
		for i, v := range elems {
			ms[l+i] = v
		}
		return ms
	case []HTML:
		l := len(s)
		ms := make(MutableSlice, l+len(elems))
		for i, v := range s {
			ms[i] = v
		}
		for i, v := range elems {
			ms[l+i] = v
		}
		return ms
	case []int:
		l := len(s)
		ms := make(MutableSlice, l+len(elems))
		for i, v := range s {
			ms[i] = v
		}
		for i, v := range elems {
			ms[l+i] = v
		}
		return ms
	case []decimal.Decimal:
		l := len(s)
		ms := make(MutableSlice, l+len(elems))
		for i, v := range s {
			ms[i] = v
		}
		for i, v := range elems {
			ms[l+i] = v
		}
		return ms
	case []bool:
		l := len(s)
		ms := make(MutableSlice, l+len(elems))
		for i, v := range s {
			ms[i] = v
		}
		for i, v := range elems {
			ms[l+i] = v
		}
		return ms
	}
	sv := reflect.ValueOf(slice)
	if sv.Kind() != reflect.Slice {
		panic(fmt.Errorf("first argument to append must be slice; have %s", typeof(slice)))
	}
	l := sv.Len()
	ms := make(MutableSlice, l+len(elems))
	for i := 0; i < l; i++ {
		ms[i] = sv.Index(i).Interface()
	}
	for i, v := range elems {
		ms[l+i] = v
	}
	return ms
}

// _base64 is the builtin function "base64".
func _base64(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

// _delete is the builtin function "delete".
func _delete(m MutableMap, key string) {
	delete(m, key)
}

// _errorf is the builtin function "errorf".
func _errorf(format string, a ...interface{}) {
	panic(fmt.Errorf(format, a...))
}

// _hex is the builtin function "hex".
func _hex(s string) string {
	return hex.EncodeToString([]byte(s))
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
		panic(errors.New("unknown hash function"))
	}
	mac := hmac.New(h, []byte(key))
	io.WriteString(mac, message)
	s := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return s
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
	case MutableSlice:
		return len(s)
	case []interface{}:
		return len(s)
	case []string:
		return len(s)
	case []HTML:
		return len(s)
	case []int:
		return len(s)
	case []decimal.Decimal:
		return len(s)
	case []bool:
		return len(s)
	case MutableMap:
		return len(s)
	case map[string]interface{}:
		return len(s)
	case map[string]string:
		return len(s)
	default:
		var rv = reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.Slice:
			return rv.Len()
		case reflect.Map:
			return rv.Len()
		case reflect.Ptr:
			if keys := structKeys(rv); keys != nil {
				return len(keys)
			}
		}
	}
	// Returning -1 the method evalCall will return an invalid argument error.
	return -1
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

// _replaceAll is the builtin function "replaceAll".
func _replaceAll(s, old, new string) string {
	return strings.Replace(s, old, new, -1)
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
func _shuffle(s interface{}) MutableSlice {
	if s == nil {
		return nil
	}
	var ms MutableSlice
	switch m := s.(type) {
	case MutableSlice:
		ms = m
	case []interface{}:
		ms = make(MutableSlice, len(m))
		copy(ms, m)
	default:
		rv := reflect.ValueOf(s)
		if rv.Kind() != reflect.Slice {
			panic(errNoSlice)
		}
		l := rv.Len()
		ms = make(MutableSlice, l)
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
func _sort(slice interface{}) MutableSlice {
	if slice == nil {
		return nil
	}
	// no reflect
	switch s := slice.(type) {
	case MutableSlice:
		if len(s) < 2 {
			return s
		}
		defer func() {
			if r := recover(); r != nil {
				panic(errors.New("no slice of string, number or bool"))
			}
		}()
		switch s[0].(type) {
		case string, HTML:
			sort.Slice(s, func(i, j int) bool {
				var ok bool
				var si, sj string
				if si, ok = s[i].(string); !ok {
					si = string(s[i].(HTML))
				}
				if sj, ok = s[j].(string); !ok {
					sj = string(s[j].(HTML))
				}
				return si < sj
			})
			return s
		case decimal.Decimal, int:
			sort.Slice(s, func(i, j int) bool {
				var ok bool
				var si, sj decimal.Decimal
				if si, ok = s[i].(decimal.Decimal); !ok {
					si = decimal.New(int64(s[i].(int)), 0)
				}
				if sj, ok = s[j].(decimal.Decimal); !ok {
					sj = decimal.New(int64(s[j].(int)), 0)
				}
				return si.LessThan(sj)
			})
			return s
		case bool:
			sort.Slice(s, func(i, j int) bool { return !s[i].(bool) })
			return s
		}
	case []string:
		ms := make(MutableSlice, len(s))
		for i := 0; i < len(s); i++ {
			ms[i] = s[i]
		}
		sort.Slice(ms, func(i, j int) bool { return ms[i].(string) < ms[j].(string) })
		return ms
	case []HTML:
		ms := make(MutableSlice, len(s))
		for i := 0; i < len(s); i++ {
			ms[i] = s[i]
		}
		sort.Slice(ms, func(i, j int) bool { return string(ms[i].(HTML)) < string(ms[j].(HTML)) })
		return ms
	case []int:
		ms := make(MutableSlice, len(s))
		for i := 0; i < len(s); i++ {
			ms[i] = s[i]
		}
		sort.Slice(ms, func(i, j int) bool { return ms[i].(int) < ms[j].(int) })
		return ms
	case []decimal.Decimal:
		ms := make(MutableSlice, len(s))
		for i := 0; i < len(s); i++ {
			ms[i] = s[i]
		}
		sort.Slice(ms, func(i, j int) bool {
			return ms[i].(decimal.Decimal).LessThan(ms[j].(decimal.Decimal))
		})
		return ms
	case []bool:
		ms := make(MutableSlice, len(s))
		for i := 0; i < len(s); i++ {
			ms[i] = s[i]
		}
		sort.Slice(ms, func(i, j int) bool { return !ms[i].(bool) })
		return ms
	}
	panic(errors.New("no slice of string, number or bool"))
}

// _sortBy is the builtin function "sortBy".
func _sortBy(slice interface{}, field string) interface{} {
	r, _ := utf8.DecodeRuneInString(field)
	if r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
		panic(errors.New("invalid field"))
	}
	if slice == nil {
		return slice
	}
	defer func() {
		if r := recover(); r != nil {
			panic(errors.New("call of sortBy on a no-struct value"))
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
		fv := v.FieldByName(field)
		if !fv.IsValid() {
			panic(fmt.Errorf("type struct has no field or method %s", field))
		}
		values[i] = fv.Interface()
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
