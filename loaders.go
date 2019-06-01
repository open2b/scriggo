// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scrigo

import (
	"strings"
)

// PackageLoader is implemented by package loaders. Given a package path, Load
// returns a *Package value or a package source as io.Reader.
//
// If the package does not exist it returns nil and nil.
// If the package exists but there was an error while loading the package, it
// returns nil and the error.
//
// If Load returns an io.Reader that implements io.Closer, the Close method
// will be called after a Read returns either EOF or an error.
type PackageLoader interface {
	Load(path string) (interface{}, error)
}

// MapStringLoader implements PackageLoader that returns the source of a
// package. Package paths and sources are respectively the keys and the values
// of the map.
type MapStringLoader map[string]string

func (r MapStringLoader) Load(path string) (interface{}, error) {
	if src, ok := r[path]; ok {
		return strings.NewReader(src), nil
	}
	return nil, nil
}

// CombinedLoaders combines more loaders in one loader. Load calls in order
// the Load methods of each loader and returns as soon as a loader returns
// a package.
type CombinedLoaders []PackageLoader

func (loaders CombinedLoaders) Load(path string) (interface{}, error) {
	for _, loader := range loaders {
		p, err := loader.Load(path)
		if p != nil || err != nil {
			return p, err
		}
	}
	return nil, nil
}

// Loaders returns a CombinedLoaders that combine loaders.
func Loaders(loaders ...PackageLoader) PackageLoader {
	return CombinedLoaders(loaders)
}

// Packages is a Loader that load packages from a map where the key is a
// package path and the value is a *Package value.
type Packages map[string]*Package

func (pp Packages) Load(path string) (interface{}, error) {
	if p, ok := pp[path]; ok {
		return p, nil
	}
	return nil, nil
}
