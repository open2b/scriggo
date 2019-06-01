// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scrigo

import (
	"strings"
)

// PackageLoader is implemented by package loaders. Load returns a predefined
// package as *PredefinedPackage or the source of a non predefined package as
// an io.Reader.
//
// If the package does not exist it returns nil and nil.
// If the package exists but there was an error while loading the package, it
// returns nil and the error.
//
// If the loader returns an io.Reader that implements io.Closer, the Close
// method will be called immediately after a Read returns either EOF or an
// error.
type PackageLoader interface {
	Load(path string) (interface{}, error)
}

// MapStringLoader implements PackageLoader for not predefined packages as a
// map with string values. Paths and sources are respectively the keys and the
// values of the map.
type MapStringLoader map[string]string

func (r MapStringLoader) Load(path string) (interface{}, error) {
	if src, ok := r[path]; ok {
		return strings.NewReader(src), nil
	}
	return nil, nil
}
