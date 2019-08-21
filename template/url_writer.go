// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

import (
	"bytes"
	"html"
	"io"
	"strings"
)

type urlWriter struct {
	path   bool
	query  bool
	addAmp bool
	isSet  bool
	quoted bool
	w      io.Writer
}

func (w *urlWriter) Init(quoted, isSet bool) {
	w.path = true
	w.query = false
	w.addAmp = false
	w.quoted = quoted
	w.isSet = isSet
}

// Write handle *ast.Show nodes in context URL.
func (w *urlWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	s := html.UnescapeString(string(p)) // TODO(Gianluca): optimize.
	sw := newStringWriter(w.w)
	if w.path {
		if strings.Contains(s, "?") {
			w.path = false
			w.addAmp = s[len(s)-1] != '?' && s[len(s)-1] != '&'
		}
		return pathEscape(sw, s, w.quoted)
	}
	if w.query {
		return queryEscape(sw, s)
	}
	panic("not w.path and not w.query...")
	return 0, nil
}

// WriteText handle the *ast.Text nodes in context URL.
func (w *urlWriter) WriteText(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	n := 0
	if !w.query {
		if bytes.ContainsAny(p, "?#") {
			if p[0] == '?' && !w.path {
				if w.addAmp {
					nn, err := io.WriteString(w.w, "&amp;")
					n += nn
					if err != nil {
						return n, err
					}
				}
				p = p[1:]
			}
			w.path = false
			w.query = true
		}
		if w.isSet && bytes.ContainsRune(p, ',') {
			w.path = true
			w.query = false
		}
	}
	nn, err := w.w.Write(p)
	n += nn
	return n, err
}
