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

// urlWriter implements an io.Writer that escapes a URL or a set of URL and
// writes it to another writer. An urlWriter is passed to a RenderFunc
// function and to a ValueRenderer Render function when the context is
// ContextAttribute or ContextUnquotedAttribute and the value of the
// attribute is an URL or a set of URL.
type urlWriter struct {
	// path reports whether it is currently in the path.
	path bool
	// query reports whether it is currently in the query string.
	query bool
	// addAmp reports whether an ampersand will be added before the next
	// written text.
	addAmp bool
	// isSet reports whether the attribute is a set of URLs.
	isSet bool
	// quoted reports whether the attribute is quoted.
	quoted bool
	// w is the io.Writer to write to.
	w io.Writer
}

// Init initializes the urlWriter and it is called when an URL attribute value
// starts.
func (w *urlWriter) Init(quoted, isSet bool) {
	w.path = true
	w.query = false
	w.addAmp = false
	w.quoted = quoted
	w.isSet = isSet
}

// WriteText is called for each template {{ value }} in a URL attribute value.
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

// WriteText is called for each template text in a URL attribute value.
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
