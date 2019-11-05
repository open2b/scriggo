// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

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

// justOneIsDefined returns true only if just one of t1 and t2 is a defined
// type.
func justOneIsDefined(t1, t2 reflect.Type) bool {
	return (isDefinedType(t1) && !isDefinedType(t2)) || (!isDefinedType(t1) && isDefinedType(t2))
}

func Zero(t reflect.Type) reflect.Value {
	if st, ok := t.(ScriggoType); ok {
		return Zero(st.Underlying())
	}
	return reflect.Zero(t)
}

func AssignableTo(x, t reflect.Type) bool {

	T, scriggoType := t.(ScriggoType)

	// T is not a Scriggo type (is a *reflect.rtype), so it's safe to call the
	// reflect method AssignableTo with T as argument.
	if !scriggoType {
		return x.AssignableTo(t)
	}

	// T is a Scriggo type.
	// x can be both a Scriggo type or a Go type.

	if x == T {
		return true
	}

	// x and T are both defined types but they are not the same: not assignable.
	if isDefinedType(x) && isDefinedType(T) {
		return false
	}

	if justOneIsDefined(x, T) {
		return x.AssignableTo(T.Underlying())
	}

	// TODO.
	return false
}
