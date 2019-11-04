// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"reflect"
)

type scriggoType struct {
	reflect.Type
	name string
	elem *scriggoType
	// Path string
	// Methods []Method
}

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

func (x scriggoType) AssignableTo(T reflect.Type) bool {

	// If both x and T are Scriggo defined types, the assignment can be done
	// only if they are the same type.
	if T, ok := T.(scriggoType); ok {
		return x == T
	}

	if T.Name() == "" && x.elem == nil {
		return x.Type.AssignableTo(T)
	}

	// x is a type defined in Scriggo and T is a type defined in Go.
	return false

}

func (st scriggoType) Elem() reflect.Type {
	return st.elem
}

func (st scriggoType) Name() string {
	switch st.Type.Kind() {
	case reflect.Slice:
		return "[]" + st.elem.Name()
	default:
		return st.name
	}

}

func (st scriggoType) String() string {
	return st.Name() // TODO
}

// Underlying returns the underlying type of the Scriggo type.
func (st scriggoType) Underlying() reflect.Type {
	return st.Type
}

// Functions

func SliceOf(t reflect.Type) reflect.Type {
	if st, ok := t.(scriggoType); ok {
		// TODO: name is not setted here, is calculated when needed. Is that ok?
		slice := newScriggoType("", reflect.SliceOf(st.Underlying()))
		slice.elem = &st
		return slice
	}
	return reflect.SliceOf(t)
}
