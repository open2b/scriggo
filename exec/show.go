//
// Copyright (c) 2017 Open2b Software Snc. All Rights Reserved.
//

package exec

import (
	"bytes"
	"fmt"
	"html"
	"io"
	"strconv"
	"strings"

	"open2b/decimal"
)

// HTMLer è implementato da qualsiasi valore che ha metodo HTML,
// il quale definisce il formato HTML per questo valore.
// I valori che implementano HTMLer vengono mostrati nel contesto
// HTML senza essere sottoposti ad escape.
type HTMLer interface {
	HTML() string
}

func showInHTMLContext(wr io.Writer, expr interface{}) error {

	var s string

	switch e := expr.(type) {
	case string:
		s = html.EscapeString(e)
	case HTML:
		s = string(e)
	case HTMLer:
		s = e.HTML()
	case int:
		s = strconv.Itoa(e)
	case decimal.Dec:
		s = e.String()
		if strings.Contains(s, ".") {
			s = strings.TrimRight(s, "0")
			if s[len(s)-1] == '.' {
				s = s[:len(s)-1]
			}
		}
	case bool:
		if e {
			s = "true"
		} else {
			s = "false"
		}
	case []string:
		st := make([]string, len(e))
		for i, v := range e {
			st[i] = html.EscapeString(v)
		}
		s = strings.Join(st, ", ")
	case []HTML:
		st := make([]string, len(e))
		for i, h := range e {
			st[i] = string(h)
		}
		s = strings.Join(st, ", ")
	case []HTMLer:
		st := make([]string, len(e))
		for i, h := range e {
			st[i] = h.HTML()
		}
		s = strings.Join(st, ", ")
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

func showInScriptContext(wr io.Writer, expr interface{}) error {
	_, err := io.WriteString(wr, toJavaScriptValue(expr))
	return err
}

func toJavaScriptValue(expr interface{}) string {

	switch e := expr.(type) {
	case string:
		return toJavaScriptString(e)
	case HTML:
		return toJavaScriptString(string(e))
	case HTMLer:
		return toJavaScriptString(e.HTML())
	case int:
		return strconv.Itoa(e)
	case decimal.Dec:
		s := e.String()
		if strings.Contains(s, ".") {
			s = strings.TrimRight(s, "0")
			if s[len(s)-1] == '.' {
				s = s[:len(s)-1]
			}
		}
		return s
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
	case []HTML:
		if e == nil {
			return "null"
		}
		var s string
		for i, t := range e {
			if i > 0 {
				s += ","
			}
			s += toJavaScriptString(string(t))
		}
		return "[" + s + "]"
	case []HTMLer:
		if e == nil {
			return "null"
		}
		var s string
		for i, t := range e {
			if i > 0 {
				s += ","
			}
			s += toJavaScriptString(t.HTML())
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
