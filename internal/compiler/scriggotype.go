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

// newScriggoDefinedType creates a new Scriggo defined type, that is a type
// created with the syntax
//
//    type Int int
//
func newScriggoDefinedType(name string, baseType reflect.Type) scriggoType {
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

func (st scriggoType) Kind() reflect.Kind {
	return st.Underlying().Kind()
}

func (st scriggoType) Elem() reflect.Type {
	if st.elem == nil {
		panic("BUG: cannot call method Elem() on a scriggoType with nil elem")
	}
	return st.elem
}

func (st scriggoType) Len() int {
	panic("not implemented") // TODO.
}

func (st scriggoType) Name() string {
	if st.name != "" {
		return st.name
	}

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

// TODO: change all calls to reflect.SliceOf to SliceOf.
func SliceOf(t reflect.Type) reflect.Type {
	if st, ok := t.(scriggoType); ok {
		slice := scriggoType{
			Type: SliceOf(st.Underlying()),
			elem: &st,
		}
		return slice
	}
	return reflect.SliceOf(t)
}

// TODO: change all calls to reflect.Zero to Zero.
func Zero(t reflect.Type) reflect.Value {
	if st, ok := t.(scriggoType); ok {
		return Zero(st.Underlying())
	}
	return reflect.Zero(t)
}
