//
// Copyright (c) 2017 Open2b Software Snc. All Rights Reserved.
//

package exec

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"open2b/decimal"
)

func showInHTMLContext(wr io.Writer, expr interface{}) error {

	var s string

	switch e := expr.(type) {
	case string:
		s = e
	case int:
		s = strconv.Itoa(e)
	case decimal.Dec:
		s = e.String()
		i := len(s) - 1
		for i > 0 && s[i] == '0' {
			i--
		}
		if s[i] == '.' {
			s = s[:i]
		} else {
			s = s[:i+1]
		}
	case bool:
		if e {
			s = "true"
		} else {
			s = "false"
		}
	case []string:
		s = strings.Join(e, ", ")
	case []int:
		buf := make([]string, len(e))
		for i, n := range e {
			buf[i] = strconv.Itoa(n)
		}
		s = strings.Join(buf, ", ")
	case []bool:
		buf := make([]string, len(e))
		for i, b := range e {
			if b {
				buf[i] = "true"
			} else {
				buf[i] = "false"
			}
		}
		s = strings.Join(buf, ", ")
	default:
		if str, ok := e.(fmt.Stringer); ok {
			s = str.String()
		}
	}

	_, err := io.WriteString(wr, s)

	return err
}

func showInAttributeContext(wr io.Writer, expr interface{}) error {
	return errors.New("Attribute context is not supported")
}

func showInScriptContext(wr io.Writer, expr interface{}) error {
	_, err := io.WriteString(wr, toJavaScriptValue(expr))
	return err
}

func toJavaScriptValue(expr interface{}) string {

	switch e := expr.(type) {
	case string:
		return toJavaScriptString(e)
	case int:
		return strconv.Itoa(e)
	case decimal.Dec:
		s := e.String()
		i := len(s) - 1
		for i > 0 && s[i] == '0' {
			i--
		}
		if s[i] == '.' {
			return s[:i]
		}
		return s[:i+1]
	case bool:
		if e {
			return "true"
		}
		return "false"
	case map[string]interface{}:
		return toJavaScriptObject(e)
	case []string:
		if e == nil {
			return "null"
		}
		var s string
		for i, t := range e {
			if i > 0 {
				s += ","
			}
			s += toJavaScriptString(t)
		}
		return "[" + s + "]"
	case []int:
		if e == nil {
			return "null"
		}
		var s string
		for i, n := range e {
			if i > 0 {
				s += ","
			}
			s += strconv.Itoa(n)
		}
		return "[" + s + "]"
	case []bool:
		if e == nil {
			return "null"
		}
		buf := make([]string, len(e))
		for i, b := range e {
			if b {
				buf[i] = "true"
			} else {
				buf[i] = "false"
			}
		}
		return "[" + strings.Join(buf, ",") + "]"
	case []map[string]interface{}:
		if e == nil {
			return "null"
		}
		buf := make([]string, len(e))
		for i, v := range e {
			buf[i] = toJavaScriptObject(v)
		}
		return "[" + strings.Join(buf, ",") + "]"
	default:
		if str, ok := e.(fmt.Stringer); ok {
			return str.String()
		}
	}

	return ""
}

const hex = "0123456789abcdef"

func toJavaScriptString(s string) string {
	if len(s) == 0 {
		return "\"\""
	}
	var b bytes.Buffer
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString("\\\\")
		case '"':
			b.WriteString("\\\"")
		case '\n':
			b.WriteString("\\n")
		case '\r':
			b.WriteString("\\r")
		case '\t':
			b.WriteString("\\t")
		case '\u2028':
			b.WriteString("\\u2028")
		case '\u2029':
			b.WriteString("\\u2029")
		default:
			if 0 <= r && r <= 31 || r == '<' || r == '>' || r == '&' {
				b.WriteString("\\x")
				b.WriteByte(hex[r>>4])
				b.WriteByte(hex[r&0xF])
			} else {
				b.WriteRune(r)
			}
		}
	}
	return "\"" + b.String() + "\""
}

func toJavaScriptObject(e map[string]interface{}) string {
	if e == nil {
		return "null"
	}
	var s string
	for k, v := range e {
		s += toJavaScriptString(k) + ":" + toJavaScriptValue(v) + ","
	}
	if len(s) > 0 {
		s = s[:len(s)-1]
	}
	return "{" + s + "}"
}
