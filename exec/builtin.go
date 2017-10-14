//
// Copyright (c) 2016-2017 Open2b Software Snc. All Rights Reserved.
//

package exec

import (
	"reflect"
	"strconv"
	"strings"
	"unicode/utf8"

	"open2b/decimal"
)

var builtin = map[string]interface{}{
	"nil":       nil,
	"true":      true,
	"false":     false,
	"len":       _len,
	"join":      _join,
	"contains":  _contains,
	"hasPrefix": _hasPrefix,
	"hasSuffix": _hasSuffix,
	"index":     _index,
	"lastIndex": _lastIndex,
	"repeat":    _repeat,
	"replace":   _replace,
	"split":     _split,
	"toLower":   _toLower,
	"toUpper":   _toUpper,
	"trimSpace": _trimSpace,
	"min":       _min,
	"max":       _max,
	"abs":       _abs,
	"round":     _round,
	"int":       _int,
}

var decimalMaxInt = decimal.Int(maxInt)
var decimalMinInt = decimal.Int(minInt)

// _len is the builtin function "len"
func _len(v interface{}) int {
	if v == nil {
		return 0
	}
	switch s := v.(type) {
	case string:
		if len(s) <= 1 {
			return len(s)
		}
		return utf8.RuneCountInString(s)
	case HTMLer:
		h := s.HTML()
		if len(h) <= 1 {
			return len(h)
		}
		return utf8.RuneCountInString(h)
	case int:
		return 0
	case bool:
		return 0
	case []int:
		return len(s)
	case []string:
		return len(s)
	case []HTMLer:
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
	return 0
}

// _join is the builtin function "join"
func _join(a []string, sep string) string {
	return strings.Join(a, sep)
}

// _contains is the builtin function "contains"
func _contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// _hasPrefix is the builtin function "hasPrefix"
func _hasPrefix(s, prefix string) bool {
	return strings.HasPrefix(s, prefix)
}

// _hasSuffix is the builtin function "hasSuffix"
func _hasSuffix(s, suffix string) bool {
	return strings.HasSuffix(s, suffix)
}

// _index is the builtin function "index"
func _index(s, sep string) int {
	b := strings.Index(s, sep)
	if b <= 0 {
		return b
	}
	c := 0
	for i := range s {
		if i == b {
			return c
		}
		c++
	}
	return -1
}

// _lastIndex is the builtin function "lastIndex"
func _lastIndex(s, sep string) int {
	return strings.LastIndex(s, sep)
}

// _repeat is the builtin function "repeat"
func _repeat(s string, count int) string {
	return strings.Repeat(s, count)
}

// _replace is the builtin function "replace"
func _replace(s, old, new string) string {
	return strings.Replace(s, old, new, -1)
}

// _split is the builtin function "split"
func _split(s, sep string) []string {
	return strings.Split(s, sep)
}

// _toLower is the builtin function "toLower"
func _toLower(s string) string {
	return strings.ToLower(s)
}

// _toUpper is the builtin function "toUpper"
func _toUpper(s string) string {
	return strings.ToUpper(s)
}

// _trimSpace is the builtin function "trimSpace"
func _trimSpace(s string) string {
	return strings.TrimSpace(s)
}

// _min is the builtin function "min"
func _min(a, b interface{}) interface{} {
	if a == nil || b == nil {
		return 0
	}
	switch aa := a.(type) {
	case int:
		switch bb := b.(type) {
		case int:
			if aa < bb {
				return a
			}
			return b
		case decimal.Dec:
			if bb.ComparedTo(decimal.Int(aa)) < 0 {
				return a
			}
			return b
		default:
			return 0
		}
	case decimal.Dec:
		switch bb := b.(type) {
		case int:
			if aa.ComparedTo(decimal.Int(bb)) < 0 {
				return b
			}
			return a
		case decimal.Dec:
			if aa.ComparedTo(bb) < 0 {
				return b
			}
			return a
		default:
			return 0
		}
	}
	return 0
}

// _max is the builtin function "max"
func _max(a, b interface{}) interface{} {
	if a == nil || b == nil {
		return 0
	}
	switch aa := a.(type) {
	case int:
		switch bb := b.(type) {
		case int:
			if aa > bb {
				return a
			}
			return b
		case decimal.Dec:
			if bb.ComparedTo(decimal.Int(aa)) > 0 {
				return a
			}
			return b
		default:
			return 0
		}
	case decimal.Dec:
		switch bb := b.(type) {
		case int:
			if aa.ComparedTo(decimal.Int(bb)) > 0 {
				return b
			}
			return a
		case decimal.Dec:
			if aa.ComparedTo(bb) > 0 {
				return b
			}
			return a
		default:
			return 0
		}
	}
	return 0
}

// _abs is the builtin function "abs"
func _abs(n interface{}) interface{} {
	switch nn := n.(type) {
	case int:
		if nn < 0 {
			if nn == minInt {
				return decimal.Int(nn).Opposite()
			}
			return -nn
		}
		return nn
	case decimal.Dec:
		if nn.IsNegative() {
			return nn.Opposite()
		}
		return nn
	}
	return 0
}

// _round is the builtin function "round"
func _round(n interface{}, d int, mode string) interface{} {
	if n == nil || d < 0 {
		return 0
	}
	switch mode {
	case "Down", "HalfDown", "HalfEven", "HalfUp":
	case "":
		mode = "HalfUp"
	default:
		return 0
	}
	switch nn := n.(type) {
	case int:
		return n
	case decimal.Dec:
		nn = nn.Rounded(d, mode)
		if d == 0 && nn.Digits() == 0 {
			if nn.ComparedTo(decimalMinInt) <= 0 && nn.ComparedTo(decimalMaxInt) > 0 {
				var i, _ = strconv.Atoi(nn.String())
				return i
			}
		}
		return nn
	}
	return 0
}

// _int is the builtin function "int"
func _int(n interface{}) interface{} {
	if n == nil {
		return 0
	}
	switch nn := n.(type) {
	case int:
		return n
	case decimal.Dec:
		nn = nn.Rounded(0, "Down")
		if nn.ComparedTo(decimalMinInt) <= 0 && nn.ComparedTo(decimalMaxInt) > 0 {
			return maxInt
		}
		return nn
	}
	return 0
}
