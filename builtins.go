// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scrigo

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

	"scrigo/ast"
)

var testSeed int64 = -1

var errNoSlice = errors.New("no slice")

const spaces = " \n\r\t\f" // https://infra.spec.whatwg.org/#ascii-whitespace

var interf = interface{}(nil)

var stringType = reflect.TypeOf("")
var htmlType = reflect.TypeOf(HTML(""))
var constantNumberType = reflect.TypeOf(ConstantNumber{})
var intType = reflect.TypeOf(0)
var int64Type = reflect.TypeOf(int64(0))
var int32Type = reflect.TypeOf(int32(0))
var int16Type = reflect.TypeOf(int16(0))
var int8Type = reflect.TypeOf(int8(0))
var uintType = reflect.TypeOf(uint(0))
var uint64Type = reflect.TypeOf(uint64(0))
var uint32Type = reflect.TypeOf(uint32(0))
var uint16Type = reflect.TypeOf(uint16(0))
var uint8Type = reflect.TypeOf(uint8(0))
var float64Type = reflect.TypeOf(float64(0))
var float32Type = reflect.TypeOf(float32(0))
var boolType = reflect.TypeOf(false)
var mapType = reflect.TypeOf(map[interface{}]interface{}(nil))
var sliceType = reflect.TypeOf([]interface{}(nil))
var bytesType = reflect.TypeOf([]byte(nil))
var interfaceType = reflect.TypeOf(&interf).Elem()
var runesType = reflect.TypeOf([]rune(nil))
var errorType = reflect.TypeOf((*error)(nil)).Elem()

var builtins = map[string]interface{}{
	"nil":    nil,
	"true":   true,
	"false":  false,
	"len":    nil,
	"append": nil,
	"delete": nil,
	"new":    nil,
	"copy":   nil,
	"make":   nil,

	"string":      stringType,
	"int":         intType,
	"int64":       int64Type,
	"int32":       int32Type,
	"int16":       int16Type,
	"int8":        int8Type,
	"uint":        uintType,
	"uint64":      uint64Type,
	"uint32":      uint32Type,
	"uint16":      uint16Type,
	"uint8":       uint8Type,
	"float64":     float64Type,
	"float32":     float32Type,
	"rune":        int32Type,
	"byte":        uint8Type,
	"bool":        boolType,
	"map":         mapType,
	"slice":       sliceType,
	"bytes":       bytesType,
	"interface{}": interfaceType,
	"error":       errorType,

	"abbreviate":  _abbreviate,
	"abs":         _abs,
	"atoi":        strconv.Atoi,
	"base64":      _base64,
	"contains":    strings.Contains,
	"errorf":      _errorf,
	"hasPrefix":   strings.HasPrefix,
	"hasSuffix":   strings.HasSuffix,
	"hex":         _hex,
	"hmac":        _hmac,
	"html":        _html,
	"index":       _index,
	"indexAny":    _indexAny,
	"itoa":        strconv.Itoa,
	"join":        strings.Join,
	"lastIndex":   _lastIndex,
	"max":         _max,
	"md5":         _md5,
	"min":         _min,
	"rand":        _rand,
	"randFloat":   _randFloat,
	"repeat":      _repeat,
	"replace":     strings.Replace,
	"replaceAll":  _replaceAll,
	"reverse":     _reverse,
	"round":       _round,
	"print":       _print,
	"printf":      _printf,
	"println":     _println,
	"sha1":        _sha1,
	"sha256":      _sha256,
	"shuffle":     _shuffle,
	"sort":        _sort,
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
func _abs(n interface{}) interface{} {
	switch x := n.(type) {
	case int:
		if x < 0 {
			n = -x
		}
	case int64:
		if x < 0 {
			n = -x
		}
	case int32:
		if x < 0 {
			n = -x
		}
	case int16:
		if x < 0 {
			n = -x
		}
	case int8:
		if x < 0 {
			n = -x
		}
	case uint, uint64, uint32, uint16, uint8:
	case float64:
		if x < 0 {
			n = -x
		}
	case float32:
		if x < 0 {
			n = -x
		}
	case ConstantNumber:
		var err error
		n, err = x.ToTyped()
		if err != nil {
			panic(err)
		}
		n = _abs(n)
	case CustomNumber:
		if x.Cmp(nil) < 0 {
			n = x.New().Neg(x)
		}
	default:
		panic(fmt.Sprintf("non-number argument in abs - %s", typeof(n)))
	}
	return n
}

// _append is the builtin function "append".
func (r *rendering) _append(node *ast.Call, n int) (reflect.Value, error) {

	if len(node.Args) == 0 {
		return reflect.Value{}, r.errorf(node, "missing arguments to append")
	}
	slice, err := r.eval(node.Args[0])
	if err != nil {
		return reflect.Value{}, err
	}
	if slice == nil && r.isBuiltin("nil", node.Args[0]) {
		return reflect.Value{}, r.errorf(node, "first argument to append must be typed slice; have untyped nil")
	}
	t := reflect.TypeOf(slice)
	if t.Kind() != reflect.Slice {
		return reflect.Value{}, r.errorf(node, "first argument to append must be slice; have %s", t)
	}
	if n == 0 {
		return reflect.Value{}, r.errorf(node, "%s evaluated but not used", node)
	}
	if n > 1 {
		return reflect.Value{}, r.errorf(node, "assignment mismatch: %d variables but 1 values", n)
	}

	m := len(node.Args) - 1
	if m == 0 {
		return reflect.ValueOf(slice), nil
	}

	typ := t.Elem()

	if s, ok := slice.([]interface{}); ok {
		var s2 []interface{}
		l, c := len(s), cap(s)
		p := 0
		if l+m <= c {
			s2 = make([]interface{}, m)
			s = s[:c:c]
		} else {
			s2 = make([]interface{}, l+m)
			copy(s2, s)
			s = s2
			p = l
		}
		for i := 1; i < len(node.Args); i++ {
			v, err := r.eval(node.Args[i])
			if err != nil {
				return reflect.Value{}, err
			}
			if n, ok := v.(ConstantNumber); ok {
				v, err = n.ToType(typ)
				if err != nil {
					if e, ok := err.(errConstantOverflow); ok {
						return reflect.Value{}, r.errorf(node.Args[i], "%s", e)
					}
					return reflect.Value{}, r.errorf(node.Args[i], "%s in argument to %s", err, node.Func)
				}
			}
			s2[p+i-1] = v
		}
		if l+m <= c {
			copy(s[l:], s2)
		}
		return reflect.ValueOf(s), nil
	}

	sv := reflect.ValueOf(slice)
	var sv2 reflect.Value
	l, c := sv.Len(), sv.Cap()
	p := 0
	if l+n <= c {
		sv2 = reflect.MakeSlice(t, m, m)
		sv = sv.Slice3(0, c, c)
	} else {
		sv2 = reflect.MakeSlice(t, l+m, l+m)
		reflect.Copy(sv2, sv)
		sv = sv2
		p = l
	}
	for i := 1; i < len(node.Args); i++ {
		v, err := r.eval(node.Args[i])
		if err != nil {
			return reflect.Value{}, err
		}
		if n, ok := v.(ConstantNumber); ok {
			v, err = n.ToType(typ)
			if err != nil {
				if e, ok := err.(errConstantOverflow); ok {
					return reflect.Value{}, r.errorf(node.Args[i], "%s", e)
				}
				return reflect.Value{}, r.errorf(node.Args[i], "%s in argument to %s", err, node.Func)
			}
		}
		sv2.Index(p + i - 1).Set(reflect.ValueOf(v))
	}
	if l+m <= c {
		reflect.Copy(sv2.Slice(l, l+m+1), sv2)
	}

	return sv, nil
}

// _base64 is the builtin function "base64".
func _base64(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
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
	default:
		// TODO (Gianluca): replace %v with the name of the variable that
		// contains s.
		panic(fmt.Errorf("invalid argument %v (type %T) for html", v, s))
	}
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

// _max is the builtin function "max".
func _max(x, y interface{}) interface{} {
	switch x := x.(type) {
	case int:
		if y, ok := y.(int); ok {
			if x < y {
				return y
			}
			return x
		}
	case int64:
		if y, ok := y.(int64); ok {
			if x < y {
				return y
			}
			return x
		}
	case int32:
		if y, ok := y.(int32); ok {
			if x < y {
				return y
			}
			return x
		}
	case int16:
		if y, ok := y.(int16); ok {
			if x < y {
				return y
			}
			return x
		}
	case int8:
		if y, ok := y.(int8); ok {
			if x < y {
				return y
			}
			return x
		}
	case uint:
		if y, ok := y.(uint); ok {
			if x < y {
				return y
			}
			return x
		}
	case uint64:
		if y, ok := y.(uint64); ok {
			if x < y {
				return y
			}
			return x
		}
	case uint32:
		if y, ok := y.(uint32); ok {
			if x < y {
				return y
			}
			return x
		}
	case uint16:
		if y, ok := y.(uint16); ok {
			if x < y {
				return y
			}
			return x
		}
	case uint8:
		if y, ok := y.(uint8); ok {
			if x < y {
				return y
			}
			return x
		}
	case float64:
		if y, ok := y.(float64); ok {
			if x < y {
				return y
			}
			return x
		}
	case float32:
		if y, ok := y.(float32); ok {
			if x < y {
				return y
			}
			return x
		}
	case CustomNumber:
		if reflect.TypeOf(x) == reflect.TypeOf(y) {
			if x.Cmp(y.(CustomNumber)) < 0 {
				return y
			}
			return x
		}
	}
	panic(fmt.Sprintf("arguments to max have different types: %s and %s", typeof(x), typeof(y)))
}

// _md5 is the builtin function "md5".
func _md5(s string) string {
	hasher := md5.New()
	hasher.Write([]byte(s))
	return hex.EncodeToString(hasher.Sum(nil))
}

// _min is the builtin function "min".
func _min(x, y interface{}) interface{} {
	switch x := x.(type) {
	case int:
		if y, ok := y.(int); ok {
			if x < y {
				return x
			}
			return y
		}
	case int64:
		if y, ok := y.(int64); ok {
			if x < y {
				return x
			}
			return y
		}
	case int32:
		if y, ok := y.(int32); ok {
			if x < y {
				return x
			}
			return y
		}
	case int16:
		if y, ok := y.(int16); ok {
			if x < y {
				return x
			}
			return y
		}
	case int8:
		if y, ok := y.(int8); ok {
			if x < y {
				return x
			}
			return y
		}
	case uint:
		if y, ok := y.(uint); ok {
			if x < y {
				return x
			}
			return y
		}
	case uint64:
		if y, ok := y.(uint64); ok {
			if x < y {
				return x
			}
			return y
		}
	case uint32:
		if y, ok := y.(uint32); ok {
			if x < y {
				return x
			}
			return y
		}
	case uint16:
		if y, ok := y.(uint16); ok {
			if x < y {
				return x
			}
			return y
		}
	case uint8:
		if y, ok := y.(uint8); ok {
			if x < y {
				return x
			}
			return y
		}
	case float64:
		if y, ok := y.(float64); ok {
			if x < y {
				return x
			}
			return y
		}
	case float32:
		if y, ok := y.(float32); ok {
			if x < y {
				return x
			}
			return y
		}
	case CustomNumber:
		if reflect.TypeOf(x) == reflect.TypeOf(y) {
			if x.Cmp(y.(CustomNumber)) < 0 {
				return x
			}
			return y
		}
	}
	panic(fmt.Sprintf("arguments to min have different types: %s and %s", typeof(x), typeof(y)))
}

// _print is the builtin function "print".
func _print(a ...interface{}) (n int, err error) {
	return fmt.Print(a...)
}

// _printf is the builtin function "printf".
func _printf(format string, a ...interface{}) (n int, err error) {
	return fmt.Printf(format, a...)
}

// _println is the builtin function "println".
func _println(a ...interface{}) (n int, err error) {
	return fmt.Println(a...)
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
func _round(x float64) float64 {
	return math.Round(x)
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
func _shuffle(s interface{}) Slice {
	if s == nil {
		return nil
	}
	var ms Slice
	switch m := s.(type) {
	case []interface{}:
		ms = make(Slice, len(m))
		copy(ms, m)
	default:
		rv := reflect.ValueOf(s)
		if rv.Kind() != reflect.Slice {
			panic(errNoSlice)
		}
		l := rv.Len()
		ms = make(Slice, l)
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
	case []byte:
		sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	case []int:
		sort.Ints(s)
	case []float64:
		sort.Float64s(s)
	case []bool:
		sort.Slice(s, func(i, j int) bool { return !s[i] })
	}
	// reflect
	sortSlice(slice)
}
