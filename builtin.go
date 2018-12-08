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

var builtins = map[string]interface{}{
	"nil":    nil,
	"true":   true,
	"false":  false,
	"len":    _len,
	"string": _string,
	"number": _number,

	"abbreviate":  _abbreviate,
	"abs":         _abs,
	"append":      _append,
	"base64":      _base64,
	"contains":    strings.Contains,
	"errorf":      _errorf,
	"hasPrefix":   strings.HasPrefix,
	"hasSuffix":   strings.HasSuffix,
	"hmac":        _hmac,
	"html":        _html,
	"index":       _index,
	"indexAny":    _indexAny,
	"int":         _int,
	"join":        strings.Join,
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
	"slice":       _slice,
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
func _append(slice interface{}, elems ...interface{}) (interface{}, error) {
	if slice == nil {
		return nil, fmt.Errorf("first argument to append must be slice; have untyped nil")
	}
	if elems == nil {
		return slice, nil
	}
	var ok bool
SameType:
	switch s := slice.(type) {
	case []string:
		l := len(s)
		sr := make([]string, l+len(elems))
		for i, s := range s {
			sr[i] = s
		}
		for i, vv := range elems {
			sr[l+i], ok = vv.(string)
			if !ok {
				break SameType
			}
		}
		return sr, nil
	case []HTML:
		l := len(s)
		sr := make([]HTML, l+len(elems))
		for i, s := range s {
			sr[i] = s
		}
		for i, vv := range elems {
			sr[l+i], ok = vv.(HTML)
			if !ok {
				break SameType
			}
		}
		return sr, nil
	case []int:
		l := len(s)
		sr := make([]int, l+len(elems))
		for i, s := range s {
			sr[i] = s
		}
		for i, vv := range elems {
			sr[l+i], ok = vv.(int)
			if !ok {
				break SameType
			}
		}
		return sr, nil
	case []decimal.Decimal:
		l := len(s)
		sr := make([]decimal.Decimal, l+len(elems))
		for i, s := range s {
			sr[i] = s
		}
		for i, vv := range elems {
			sr[l+i], ok = vv.(decimal.Decimal)
			if !ok {
				break SameType
			}
		}
		return sr, nil
	case []bool:
		l := len(s)
		sr := make([]bool, l+len(elems))
		for i, s := range s {
			sr[i] = s
		}
		for i, vv := range elems {
			sr[l+i], ok = vv.(bool)
			if !ok {
				break SameType
			}
		}
		return sr, nil
	}
	st := reflect.TypeOf(slice)
	if st.Kind() != reflect.Slice {
		return nil, fmt.Errorf("first argument to append must be slice; have %s", typeof(slice))
	}
	sv := reflect.ValueOf(slice)
	l := sv.Len()
	sr := make([]interface{}, l+len(elems))
	for i := 0; i < l; i++ {
		sr[i] = sv.Index(i).Interface()
	}
	for i, vv := range elems {
		if vv == nil {
			return nil, fmt.Errorf("use of untyped nil in slice")
		}
		sr[l+i] = vv
	}
	return sr, nil
}

// _base64 is the builtin function "base64".
func _base64(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

// _errorf is the builtin function "errorf".
func _errorf(format string, a ...interface{}) (interface{}, error) {
	return nil, fmt.Errorf(format, a...)
}

// _hmac is the builtin function "hmac".
func _hmac(hasher, message, key string) (string, error) {
	var h func() hash.Hash
	switch hasher {
	case "MD5":
		h = md5.New
	case "SHA-1":
		h = sha1.New
	case "SHA-256":
		h = sha256.New
	default:
		return "", errors.New("unknown hash function")
	}
	mac := hmac.New(h, []byte(key))
	io.WriteString(mac, message)
	s := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return s, nil
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

// _number is the builtin function "number".
func _number(d decimal.Decimal) decimal.Decimal {
	return d
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
func _reverse(s interface{}) (interface{}, error) {
	if s == nil {
		return s, nil
	}
	rv := reflect.ValueOf(s)
	if rv.Kind() != reflect.Slice {
		return nil, errNoSlice
	}
	l := rv.Len()
	if l <= 1 {
		return s, nil
	}
	rt := reflect.TypeOf(s)
	rvc := reflect.MakeSlice(rt, l, l)
	reflect.Copy(rvc, rv)
	sc := rvc.Interface()
	swap := reflect.Swapper(sc)
	for i, j := 0, l-1; i < j; i, j = i+1, j-1 {
		swap(i, j)
	}
	return sc, nil
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
func _shuffle(s interface{}) (interface{}, error) {
	if s == nil {
		return nil, nil
	}
	rv := reflect.ValueOf(s)
	if rv.Kind() != reflect.Slice {
		return nil, errNoSlice
	}
	l := rv.Len()
	if l < 2 {
		return s, nil
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
	return s2, nil
}

// _slice is the builtin function "slice".
func _slice(elems ...interface{}) (interface{}, error) {
	if elems == nil {
		return nil, fmt.Errorf("use of untyped nil in slice")
	}
	if len(elems) == 0 {
		return []interface{}(nil), nil
	}
	var ok bool
SameType:
	switch elems[0].(type) {
	case string:
		slice := make([]string, len(elems))
		for i, vv := range elems {
			slice[i], ok = vv.(string)
			if !ok {
				break SameType
			}
		}
		return slice, nil
	case HTML:
		slice := make([]HTML, len(elems))
		for i, vv := range elems {
			slice[i], ok = vv.(HTML)
			if !ok {
				break SameType
			}
		}
		return slice, nil
	case int:
		slice := make([]int, len(elems))
		for i, vv := range elems {
			slice[i], ok = vv.(int)
			if !ok {
				break SameType
			}
		}
		return slice, nil
	case decimal.Decimal:
		slice := make([]decimal.Decimal, len(elems))
		for i, vv := range elems {
			slice[i], ok = vv.(decimal.Decimal)
			if !ok {
				break SameType
			}
		}
		return slice, nil
	case bool:
		slice := make([]bool, len(elems))
		for i, vv := range elems {
			slice[i], ok = vv.(bool)
			if !ok {
				break SameType
			}
		}
		return slice, nil
	}
	for _, vv := range elems {
		if vv == nil {
			return nil, fmt.Errorf("use of untyped nil in slice")
		}
	}
	return elems, nil
}

// _sort is the builtin function "sort".
func _sort(slice interface{}) (interface{}, error) {
	if slice == nil {
		return slice, nil
	}
	// no reflect
	switch s := slice.(type) {
	case []string:
		if len(s) <= 1 {
			return s, nil
		}
		s2 := make([]string, len(s))
		copy(s2, s)
		sort.Strings(s2)
		return s2, nil
	case []int:
		if len(s) <= 1 {
			return s, nil
		}
		s2 := make([]int, len(s))
		copy(s2, s)
		sort.Ints(s2)
		return s2, nil
	case []bool:
		if len(s) <= 1 {
			return s, nil
		}
		s2 := make([]bool, len(s))
		copy(s2, s)
		sort.Slice(s2, func(i, j int) bool { return !s2[i] })
		return s2, nil
	}
	return nil, fmt.Errorf("no slice of string, int or bool")
}

// _sortBy is the builtin function "sortBy".
func _sortBy(slice interface{}, field string) (s interface{}, err error) {
	r, _ := utf8.DecodeRuneInString(field)
	if r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
		return nil, errors.New("invalid field")
	}
	if slice == nil {
		return slice, nil
	}
	defer func() {
		if r := recover(); r != nil {
			err = errors.New("call of sortBy on a no-struct value")
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
			return nil, fmt.Errorf("type struct has no field or method %s", field)
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
			return slice, nil
		}
		vv := make([]string, size)
		for i := 0; i < size; i++ {
			vv[i] = values[i].(Stringer).String()
		}
		f = func(i, j int) bool { return vv[i] < vv[j] }
	case Numberer:
		if size <= 1 {
			return slice, nil
		}
		vv := make([]decimal.Decimal, size)
		for i := 0; i < size; i++ {
			vv[i] = values[i].(Numberer).Number()
		}
		f = func(i, j int) bool { return vv[i].Cmp(vv[j]) < 0 }
	case string:
		if size <= 1 {
			return slice, nil
		}
		f = func(i, j int) bool { return values[i].(string) < values[j].(string) }
	case int:
		if size <= 1 {
			return slice, nil
		}
		f = func(i, j int) bool { return values[i].(int) < values[j].(int) }
	case bool:
		if size <= 1 {
			return slice, nil
		}
		f = func(i, j int) bool { return !values[i].(bool) }
	}
	rv2 := reflect.MakeSlice(rv.Type(), size, size)
	reflect.Copy(rv2, rv)
	s2 := rv2.Interface()
	sort.Slice(s2, f)
	return s2, nil
}

// _string is the builtin function "string".
func _string(s string) string {
	return s
}
