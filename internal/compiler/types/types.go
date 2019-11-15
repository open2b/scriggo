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

// Types allows to create and manage types and values, as the reflect package
// does, for a specific compilation and for both the types compiled by Scriggo
// and by gc.
type Types struct {
	// funcParams holds a list of parameters list for both input (index 0) and
	// output (index 1) parameters. This avoids the duplication of the
	// parameters list, that is both unefficient and wrong (without that, two
	// indentical functions declared in two parts of the code would have two
	// different parameters lists, making them 'different' when compared by Go).
	funcParams [2][]*[]reflect.Type

	// structFieldsLists avoid the creation of two different structTypes with
	// the same struct fields.
	structFieldsLists []*map[int]reflect.StructField
}

// NewTypes returns a new instance of Types.
func NewTypes() *Types {
	return &Types{}
}

// A ScriggoType represents a type compiled by Scriggo from a type definition or
// a composite type literal with at least one element with a Scriggo type.
type ScriggoType interface {
	reflect.Type

	// Underlying returns the reflect implementation of the reflect.Type, so
	// it's safe to pass the returned value to the functions of the reflect
	// package.
	Underlying() reflect.Type
}

// isDefinedType reports whether t is a defined type.
func isDefinedType(t reflect.Type) bool {
	return t.Name() != ""
}

// New behaves like reflect.New except when typ is a Scriggo type; in such case
// is returned an instance of the underlying type created with a reflect.New
// call.
func (types *Types) New(typ reflect.Type) reflect.Value {
	if st, ok := typ.(ScriggoType); ok {
		return types.New(st.Underlying())
	}
	return reflect.New(typ)
}

// Zero is equivalent to reflect.Zero. If t is a Scriggo type it returns the
// zero of the underlying type.
func (types *Types) Zero(t reflect.Type) reflect.Value {
	if st, ok := t.(ScriggoType); ok {
		return types.Zero(st.Underlying())
	}
	return reflect.Zero(t)
}

// AssignableTo is equivalent to reflect's AssignableTo.
func (types *Types) AssignableTo(x, u reflect.Type) bool {

	typ, isScriggoType := u.(ScriggoType)

	// T is not a Scriggo type (is a *reflect.rtype), so it's safe to call the
	// reflect method AssignableTo with T as argument.
	if !isScriggoType {
		return x.AssignableTo(u)
	}

	// typ is a Scriggo type.
	// x can be both a Scriggo type or a Go type.

	// The type is the same so x is assignable to typ.
	if x == typ {
		return true
	}

	xIsDefined := isDefinedType(x)
	tIsDefined := isDefinedType(typ)

	// x and typ are both defined types but they are not the same: not
	// assignable.
	if xIsDefined && tIsDefined {
		return false
	}

	// Just one is defined.
	if xIsDefined != tIsDefined {
		return x.AssignableTo(typ.Underlying())
	}

	// TODO: any other cases missing? Or it's ok to return 'false' here?
	return false
}

// ConvertibleTo is equivalent to reflect's ConvertibleTo.
func (types *Types) ConvertibleTo(x, u reflect.Type) bool {
	if x, ok := x.(ScriggoType); ok {
		return types.ConvertibleTo(x.Underlying(), u)
	}
	if u, ok := u.(ScriggoType); ok {
		return types.ConvertibleTo(x, u.Underlying())
	}
	return x.ConvertibleTo(u) // rtype method
}

// Implements behaves like x.Implements(u) except when one of x or u is a
// Scriggo type.
func (types *Types) Implements(x, u reflect.Type) bool {

	// x is a Scriggo type,
	// u (the interface type) can be both a Scriggo type or a Go type.
	if st, ok := x.(ScriggoType); ok {
		return st.Implements(u)
	}

	// x is a Go type,
	// u (the interface type) is a Scriggo type.
	if st, ok := u.(ScriggoType); ok {
		return x.Implements(st.Underlying())
	}

	// x is a Go type,
	// u (the interface type) is a Go type.
	return x.Implements(u) // rtype method

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
