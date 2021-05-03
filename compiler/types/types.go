// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package types implements functions and types to represent and work with
// non-native types.
package types

import (
	"fmt"
	"reflect"
)

// Types allows to create and manage types and values, as the reflect package
// does, for a specific compilation and for both the native and non-native
// types.
type Types struct {
	// funcParamsStore provides the function parameters necessary to create
	// funcTypes.
	funcParamsStore funcParams

	// structFieldsLists avoid the creation of two different structTypes with
	// the same struct fields.
	structFieldsLists []*map[int]reflect.StructField
}

// NewTypes returns a new instance of Types.
func NewTypes() *Types {
	return &Types{}
}

// A Type represents a non-native type references in source as type definition
// or a composite type literal with at least one element with a non-native
// type.
type Type interface {
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

// New behaves like reflect.New except when typ is a non-native type; in such
// case it returns an instance of the underlying type created with a
// reflect.New call.
func (types *Types) New(typ reflect.Type) reflect.Value {
	if st, ok := typ.(Type); ok {
		return types.New(st.Underlying())
	}
	return reflect.New(typ)
}

// Zero is equivalent to reflect.Zero. If t is a non-native type it returns
// the zero of the underlying type.
func (types *Types) Zero(t reflect.Type) reflect.Value {
	if st, ok := t.(Type); ok {
		return types.Zero(st.Underlying())
	}
	return reflect.Zero(t)
}

// AssignableTo is equivalent to reflect's AssignableTo.
func (types *Types) AssignableTo(x, u reflect.Type) bool {

	typ, ok := u.(Type)

	// u is a native type (is implemented by the reflect package), so it's
	// safe to call the reflect method AssignableTo with u as argument.
	if !ok {
		return x.AssignableTo(u)
	}

	// typ is a non-native type.
	// x can be both a native or non-native type.

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
	if x, ok := x.(Type); ok {
		return types.ConvertibleTo(x.Underlying(), u)
	}
	if u, ok := u.(Type); ok {
		return types.ConvertibleTo(x, u.Underlying())
	}
	return x.ConvertibleTo(u) // x is implemented by the reflect package.
}

// Implements behaves like x.Implements(u) except when one of x or u is a
// non-native type.
func (types *Types) Implements(x, u reflect.Type) bool {

	// x is a non-native type, u (the interface type) can be both a native or
	// non-native type.
	if st, ok := x.(Type); ok {
		return st.Implements(u)
	}

	// x is a native type, u (the interface type) is a non-native type.
	if st, ok := u.(Type); ok {
		return x.Implements(st.Underlying())
	}

	// x is a gc compiled type, u (the interface type) is a gc compiled type.
	return x.Implements(u) // x is implemented by the reflect package.

}

// Runtime returns a Runtime.
func (types *Types) Runtime() Runtime {
	return Runtime{types}
}

// TypeOf returns the type of v.
func (types *Types) TypeOf(v reflect.Value) reflect.Type {
	if !v.IsValid() {
		return nil
	}
	if p, ok := v.Interface().(emptyInterfaceProxy); ok {
		return p.sign
	}
	return v.Type()
}

// TODO: this function will be removed when the development of this package is
//  concluded.
func assertNativeType(t reflect.Type) {
	if _, ok := t.(Type); ok {
		panic(fmt.Errorf("%v is a non-native type!", t))
	}
}

// TODO: every call to a reflect function in the compiler should be checked and
//  eventually converted to a call to a function of this package.
