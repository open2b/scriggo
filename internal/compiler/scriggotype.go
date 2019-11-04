// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"reflect"
)

// newScriggoType creates a new type defined in Scriggo with the syntax
//
//     type Int int
//
func newScriggoType(name string, baseType reflect.Type) scriggoType {
	return scriggoType{
		Type: baseType,
		name: name,
	}
}

type scriggoType struct {
	reflect.Type
	name string
	// Path string
	// Methods []Method
}

func (x scriggoType) AssignableTo(T reflect.Type) bool {

	// If both x and T are Scriggo defined types, the assignment can be done
	// only if they are the same type.
	if T, ok := T.(scriggoType); ok {
		return x == T
	}

	if T.Name() == "" {
		return x.Type.AssignableTo(T)
	}

	// x is a type defined in Scriggo and T is a type defined in Go.
	return false

}

// Underlying returns the underlying type of the Scriggo type.
func (st scriggoType) Underlying() reflect.Type {
	return st.Type
}

func (st scriggoType) String() string {
	return st.name // TODO
}

func (st scriggoType) Name() string {
	return st.name
}
