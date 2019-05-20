// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package native

import "reflect"

// TODO (Gianluca): find a better name.
// TODO (Gianluca): this identifier must be accessed from outside as
// "scrigo.Package" or something similar.
type GoPackage struct {
	Name         string
	Declarations map[string]interface{}
}

type pkgConstant struct {
	value interface{}
	typ   reflect.Type // nil for untyped constants.
}

// Constant returns a constant with given value and type. Can be used in Scrigo
// packages definition. typ is nil for untyped constants.
func Constant(value interface{}, typ reflect.Type) pkgConstant {
	return pkgConstant{value, typ}
}
