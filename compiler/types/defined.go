// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"reflect"
)

// definedType represents a non-native defined type, where the underlying type
// can be both a native or non-native type.
type definedType struct {
	// The embedded reflect.Type can be both a reflect.Type implemented by the
	// package "reflect" or a types.Type. In the other implementations of
	// Type the embedded reflect.Type is always implemented by the package
	// "reflect".
	reflect.Type

	name string
}

// DefinedOf returns the defined type with the given name and underlying type.
// For example, if n is "Int" and k represents int, DefinedOf(n, k) represents
// the type Int declared with 'type Int int'.
func (types *Types) DefinedOf(name string, underlyingType reflect.Type) reflect.Type {
	if name == "" {
		panic("BUG: name cannot be empty")
	}
	return definedType{Type: underlyingType, name: name}
}

func (x definedType) Name() string {
	return x.name
}

// AssignableTo is equivalent to reflect's AssignableTo.
func (x definedType) AssignableTo(u reflect.Type) bool {

	// Both x and u are non-native defined types: x is assignable to u only if
	// they are the same type.
	if T, ok := u.(definedType); ok {
		return x == T
	}

	// x is a non-native defined type and u is not a defined type: x is
	// assignable to u only if the underlying type of x is assignable to u.
	if !isDefinedType(u) {
		return x.Type.AssignableTo(u)
	}

	// x is a non-native defined type and u is a Go defined type: assignment
	// is always impossible.
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

// Underlying implements the interface runtime.Wrapper.
func (x definedType) Underlying() reflect.Type {
	if st, ok := x.Type.(Type); ok {
		return st.Underlying()
	}
	assertNativeType(x.Type)
	return x.Type
}

// Unwrap implements the interface runtime.Wrapper.
func (x definedType) Unwrap(v reflect.Value) (reflect.Value, bool) { return unwrap(x, v) }

// Wrap implements the interface runtime.Wrapper.
func (x definedType) Wrap(v reflect.Value) reflect.Value { return wrap(x, v) }
