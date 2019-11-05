// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import "reflect"

type definedType struct {
	reflect.Type

	name string
}

// DefinedOf creates a new Scriggo defined type, that is a type
// created with the syntax
//
//    type Int int
//
func DefinedOf(name string, baseType reflect.Type) reflect.Type {
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

func (x definedType) String() string {
	// For defined types the string representation is exactly the name of the
	// type; the internal structure of the type is hidden.
	// TODO: verify that this is correct.
	return x.name
}

func (x definedType) Underlying() reflect.Type {
	return x.Type
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

func (x definedType) MethodByName(string) (reflect.Method, bool) {
	// TODO.
	return reflect.Method{}, false
}
