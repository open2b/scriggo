// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

import (
	"html"
	"io"
	"reflect"
	"sort"

	"scriggo"
	"scriggo/builtins"
	"scriggo/vm"
)

func Builtins() *scriggo.Package {

	main := builtins.Main()

	// html
	main.Declarations["html"] = reflect.TypeOf(HTML(""))

	// escape
	builtinEscape := main.Declarations["escape"].(func(*vm.Env, string) string)
	main.Declarations["escape"] = func(env *vm.Env, s string) HTML {
		return HTML(builtinEscape(env, s))
	}

	// sort
	builtinSort := main.Declarations["sort"].(func(interface{}))
	main.Declarations["sort"] = func(slice interface{}) {
		if s, ok := slice.([]HTML); ok {
			sort.Slice(s, func(i, j int) bool { return string(s[i]) < string(s[j]) })
		} else {
			builtinSort(slice)
		}
	}

	return main
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
		err = attributeEscape(w, html.UnescapeString(string(h)), true)
		//} else {
		//	err = r.renderInAttributeURL(w, value, node, urlstate, true)
		//}
	case ContextUnquotedAttribute:
		//if urlstate == nil {
		err = attributeEscape(w, html.UnescapeString(string(h)), false)
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
