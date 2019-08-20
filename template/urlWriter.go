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
	w      io.Writer
}

// Write handle *ast.Show nodes in context URL.
func (w *urlWriter) Write(p []byte) (int, error) {
	s := html.UnescapeString(string(p))
	sw := newStringWriter(w.w)
	if w.path {
		if strings.Contains(s, "?") {
			w.path = false
			w.addAmp = s[len(s)-1] != '?' && s[len(s)-1] != '&'
		}
		err := pathEscape(sw, s, false)
		if err != nil {
			return 0, err
		}
	} else if w.query {
		err := queryEscape(sw, s)
		if err != nil {
			return 0, err
		}
	}
	return len(p), nil
}

// WriteText handle the *ast.Text nodes in context URL.
func (w *urlWriter) WriteText(p []byte) (int, error) {
	text := []byte(html.UnescapeString(string(p)))
	if !w.query {
		if bytes.ContainsAny(text, "?#") {
			if text[0] == '?' && !w.path {
				if w.addAmp {
					_, err := io.WriteString(w.w, "&amp;")
					if err != nil {
						return 0, err
					}
				}
				text = text[1:]
			}
			w.path = false
			w.query = true
		}
		if w.isSet && bytes.IndexByte(text, ',') >= 0 { // TODO(Gianluca): why not use bytes.Contains?
			w.path = true
			w.query = false
		}
	}
	return w.w.Write(text)
}

func (w *urlWriter) Reset() {
	w.path = true
	w.query = false
	w.addAmp = false
	// TODO(Gianluca): isSet should be node.Attribute == "srcset". See
	// https://github.com/open2b/commerceready/blob/18d101b986d8ae53cf316a66ba8b4e57d2849242/reports-cgi/open2b/template/rendering.go#L141-L149
	w.isSet = false
}

type stringWriter interface {
	Write(b []byte) (int, error)
	WriteString(s string) (int, error)
}
