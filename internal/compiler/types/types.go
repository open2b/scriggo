// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// package types implements functions and types to represent and work with
// Scriggo types.
//
// In the documentation of this package the expression 'Go type' refers to a
// type that can be represented by the package 'reflect' implementation of
// 'reflect.Type'.
package types

import (
	"fmt"
	"reflect"
)

// A Types helps you work with Scriggo types and Go types.
type Types struct {
	// funcParams holds a list of parameters list for both input (index 0) and
	// output (index 1) parameters. This avoids the duplication of the
	// parameters list, that is both unefficient and wrong (without that, two
	// indentical functions declared in two parts of the code would have two
	// different parameters lists, making them 'different' when compared by Go).
	funcParams [2][]*[]reflect.Type
}

// NewTypes returns a new value with type Types than can be used to build and
// operate with Scriggo types and Go types.
func NewTypes() *Types {
	return &Types{}
}

// A ScriggoType can represent both a type defined in Scriggo and a composite
// type that "contains" a type defined in Scriggo. Any other type can be
// represented using a "real" reflect.Type (i.e. the reflect.Type implementation
// from the 'reflect' package.).
type ScriggoType interface {
	reflect.Type

	// Underlying always returns the reflect implementation of the reflect.Type,
	// so it's safe to pass the returned value to the functions exported by the
	// reflect package.
	Underlying() reflect.Type
}

// isDefinedType reports whether t is a defined type.
func isDefinedType(t reflect.Type) bool {
	return t.Name() != ""
}

// justOneIsDefined returns true only if just one of t1 and t2 is a defined type.
func justOneIsDefined(t1, t2 reflect.Type) bool {
	return (isDefinedType(t1) && !isDefinedType(t2)) || (!isDefinedType(t1) && isDefinedType(t2))
}

// New behaves like reflect.New expect when typ is a Scriggo type; in such case
// is returned a 'new' instance of the underlying Go type.
func (types *Types) New(typ reflect.Type) reflect.Value {
	if st, ok := typ.(ScriggoType); ok {
		return types.New(st.Underlying())
	}
	return reflect.New(typ)
}

// Zero behaves like reflect.Zero except when t is a Scriggo type; in such case
// instead of returning the zero of the Scriggo type is returned the zero of the
// underlying type.
func (types *Types) Zero(t reflect.Type) reflect.Value {
	if st, ok := t.(ScriggoType); ok {
		return types.Zero(st.Underlying())
	}
	return reflect.Zero(t)
}

// AssignableTo behaves like x.AssignableTo(t) except when t is a Scriggo type.
func (types *Types) AssignableTo(x, t reflect.Type) bool {

	T, isScriggoType := t.(ScriggoType)

	// T is not a Scriggo type (is a *reflect.rtype), so it's safe to call the
	// reflect method AssignableTo with T as argument.
	if !isScriggoType {
		return x.AssignableTo(t)
	}

	// T is a Scriggo type.
	// x can be both a Scriggo type or a Go type.

	// The type is the same so x is assignable to T.
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

	// TODO: any other cases missing? Or it's ok to return 'false' here?
	return false
}

// ConvertibleTo behaves like x.ConvertibleTo(u) except when one of x or u is a
// Scriggo type.
func (types *Types) ConvertibleTo(x, u reflect.Type) bool {
	if x, ok := x.(ScriggoType); ok {
		return types.ConvertibleTo(x.Underlying(), u)
	}
	if u, ok := u.(ScriggoType); ok {
		return types.ConvertibleTo(x, u.Underlying())
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

// TODO: every call to a reflect function in the compiler should be checked and
// eventually converted to a call to a function of this package.

// TODO: using a pointer to a map/slice etc.. as a field makes comparisons
// return false when two identical Scriggo types are created in two different
// moments.
