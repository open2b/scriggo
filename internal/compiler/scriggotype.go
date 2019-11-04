// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"reflect"
)

type ScriggoType interface {
	reflect.Type
	Underlying() reflect.Type
}

func isDefinedType(t reflect.Type) bool {
	return t.Name() != ""
}

func Zero(t reflect.Type) reflect.Value {
	if st, ok := t.(ScriggoType); ok {
		return Zero(st.Underlying())
	}
	return reflect.Zero(t)
}
