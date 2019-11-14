// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"reflect"
)

// A definedType is a type defined in Scriggo with the syntax
//
//      type Name subType
//
//  subType can be both a Scriggo type or a Go Type.
//
type definedType struct {
	// The embedded reflect.Type can be both a reflect.Type implemented by the
	// package "reflect" or a ScriggoType. In the other implementations of
	// ScriggoType the embedded reflect.Type is always a Go type.
	reflect.Type

	// name is the name of the defined type.
	//
	//     type name subtype
	//
	name string
}

// DefinedOf creates a new Scriggo defined type, that is a type
// created with the syntax
//
//    type Int int
//
func (types *Types) DefinedOf(name string, baseType reflect.Type) reflect.Type {
	if name == "" {
		panic("BUG: name cannot be empty")
	}
	return definedType{
		Type: baseType,
		name: name,
	}
}

func (x definedType) Name() string {
	return x.name
}

func (x definedType) AssignableTo(T reflect.Type) bool {

	// Both x and T are Scriggo defined types: x is assignable to T only if they
	// are the same type.
	if T, ok := T.(definedType); ok {
		return x == T
	}

	// x is a Scriggo defined type and T is not a defined type: x is assignable
	// to T only if the underlying type of x is assignable to T.
	if !isDefinedType(T) {
		return x.Type.AssignableTo(T)
	}

	// x is a Scriggo defined type and T is a Go defined type: assignment is
	// always impossible.
	return false

}

func (x definedType) Implements(u reflect.Type) bool {
	if u.Kind() != reflect.Interface {
		panic("expected reflect.Interface")
	}
	// TODO: currently methods definition is not supported, so every defined
	// type implements only the empty interface.
	return u.NumMethod() == 0
}

func (x definedType) MethodByName(string) (reflect.Method, bool) {
	// TODO.
	return reflect.Method{}, false
}

func (x definedType) String() string {
	// For defined types the string representation is exactly the name of the
	// type; the internal structure of the type is hidden.
	// TODO: verify that this is correct.
	return x.name
}

// Underlying implement the interface runtime.Wrapper.
func (x definedType) Underlying() reflect.Type {
	if st, ok := x.Type.(ScriggoType); ok {
		return st.Underlying()
	}
	assertNotScriggoType(x.Type)
	return x.Type
}

// Unwrap implement the interface runtime.Wrapper.
func (x definedType) Unwrap(v reflect.Value) (reflect.Value, bool) { return unwrap(x, v) }

// Wrap implement the interface runtime.Wrapper.

func (x definedType) Wrap(v reflect.Value) reflect.Value { return wrap(x, v) }
