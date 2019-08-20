// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

import "io"

type urlWriter struct {
	path  bool
	query bool
	amper bool
	w     io.Writer
}

func (w *urlWriter) Write(p []byte) (int, error) {
	// TODO(Gianluca):
	return w.w.Write(p)
}

func (w *urlWriter) WriteText(p []byte) (int, error) {
	// TODO(Gianluca):
	return w.w.Write(p)
}

func (w *urlWriter) Reset() {
	w.path = false
	w.query = false
	w.amper = false
}
