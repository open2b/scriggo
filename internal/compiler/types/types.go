// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"fmt"
	"reflect"
)

// A ScriggoType represents both a type defined in Scriggo and a composite type
// that "contains" a type defined in Scriggo. Any other type can be represented
// using a "real" reflect.Type (*reflect.rtype).
type ScriggoType interface {
	reflect.Type
	// Underlying always returns the reflect implementation of the reflect.Type,
	// so it's safe to pass the returned value to the functions exported by the
	// reflect package.
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

// TODO: every call to a reflect function in the compiler should be checked and
// eventually converted to a call to a function of this package.

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

func ConvertibleTo(x, u reflect.Type) bool {
	if x, ok := x.(ScriggoType); ok {
		return ConvertibleTo(x.Underlying(), u)
	}
	if u, ok := u.(ScriggoType); ok {
		return ConvertibleTo(x, u.Underlying())
	}
	return x.ConvertibleTo(u) // rtype method
}

// TODO: this function will be removed when the development of this package is
// concluded.
func assertNotScriggoType(t reflect.Type) {
	if _, ok := t.(ScriggoType); ok {
		panic(fmt.Errorf("%v is a Scriggo type!", t))
	}
}
