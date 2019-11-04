// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"reflect"
)

// scriggoType represents a Scriggo type, which can be both:
//
// 	- a type defined in Scriggo
//  - a non-defined type that contains some types defined in Scriggo
//
// For example:
//
//		type Int int
//
// creates a new Scriggo type that is a defined-type with underlying type int.
type scriggoType struct {
	reflect.Type

	// definedName is the name of the defined type.
	//
	//		type definedName int
	//
	// If the Scriggo type is not defined (i.e. is a composite type) then
	// definedName is empty.
	definedName string

	elem *scriggoType    // slices, arrays, maps and pointers
	in   *[]reflect.Type // for functions
	out  *[]reflect.Type // for functions

	// Path string
	// Methods []Method
}

// newScriggoDefinedType creates a new Scriggo defined type, that is a type
// created with the syntax
//
//    type Int int
//
func newScriggoDefinedType(definedName string, baseType reflect.Type) scriggoType {
	if definedName == "" {
		panic("BUG: definedName cannot be empty")
	}
	return scriggoType{
		Type:        baseType,
		definedName: definedName,
	}
}

func isDefinedType(t reflect.Type) bool {
	return t.Name() != ""
}

// isCompositeType reports whether the Scriggo type is a composite type, that is
// a type without a name composed by other Scriggo types.
// Some examples are:
//
//		[]Int
//		map[String]int
//  	func(Int) int
//
func (st scriggoType) isCompositeType() bool {
	isComposite := st.elem != nil || st.in != nil || st.out != nil
	if isComposite && isDefinedType(st) {
		panic("BUG: a Scriggo type cannot be both composite and defined")
	}
	return isComposite
}

func (x scriggoType) AssignableTo(T reflect.Type) bool {

	// If both x and T are Scriggo defined types, the assignment can be done
	// only if they are the same type.
	if T, ok := T.(scriggoType); ok {
		if isDefinedType(x) && isDefinedType(T) {
			return x == T
		}
	}

	// x is a defined type byt it's not a composite type (it just have a name,
	// internally is a Go type) and T is not a defined type.
	if !x.isCompositeType() && !isDefinedType(T) {
		return x.Type.AssignableTo(T)
	}

	return false

}

func (st scriggoType) In(i int) reflect.Type {
	if st.in != nil {
		return (*st.in)[i]
	}
	return st.Underlying().In(i)
}

func (st scriggoType) Out(i int) reflect.Type {
	if st.out != nil {
		return (*st.out)[i]
	}
	return st.Underlying().Out(i)
}

func (st scriggoType) Kind() reflect.Kind {
	return st.Underlying().Kind()
}

func (st scriggoType) Elem() reflect.Type {
	if st.elem != nil {
		return *st.elem
	}
	return st.Type.Elem()
}

func (st scriggoType) Len() int {
	panic("not implemented") // TODO.
}

// Name returns the type's name within its package for a defined type.
// For other (non-defined) types it returns the empty string.
func (st scriggoType) Name() string {
	return st.definedName
}

// TODO: add support for packages path.
func (st scriggoType) String() string {
	if st.definedName != "" {
		return st.definedName
	}
	switch st.Type.Kind() {
	case reflect.Slice:
		return "[]" + st.elem.Name()
	case reflect.Func:
		s := "func("
		for i, t := range *st.in {
			s += t.String()
			if i != len(*st.in)-1 {
				s += ", "
			}
		}
		s += ") "
		if len(*st.out) >= 2 {
			s += "("
		}
		for i, t := range *st.out {
			s += t.String()
			if i != len(*st.out)-1 {
				s += ", "
			}
		}
		if len(*st.out) >= 2 {
			s += ")"
		}
		return s
	case reflect.Array, reflect.Ptr, reflect.Struct, reflect.Map:
		panic("TODO: not implemented") // TODO(Gianluca): to implement.
	default:
		return st.definedName
	}
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

func FuncOf(in, out []reflect.Type, variadic bool) reflect.Type {

	// First: check if this function contains a Scriggo type in its parameters.
	// If not, such function can be created with reflect.FuncOf without any
	// problem.
	isScriggoType := false
	for _, t := range in {
		if _, ok := t.(scriggoType); ok {
			isScriggoType = true
			break
		}
	}
	for _, t := range out {
		if _, ok := t.(scriggoType); ok {
			isScriggoType = true
			break
		}
	}

	if isScriggoType {

		inBase := make([]reflect.Type, len(in))
		outBase := make([]reflect.Type, len(out))

		for i := range in {
			if st, ok := in[i].(scriggoType); ok {
				inBase[i] = st.Underlying()
			} else {
				inBase[i] = in[i]
			}
		}
		for i := range out {
			if st, ok := out[i].(scriggoType); ok {
				outBase[i] = st.Underlying()
			} else {
				outBase[i] = out[i]
			}
		}

		return scriggoType{
			in:   &in,
			out:  &out,
			Type: FuncOf(inBase, outBase, variadic),
		}

	}

	return reflect.FuncOf(in, out, variadic)

}

// TODO: change all calls to reflect.Zero to Zero.
func Zero(t reflect.Type) reflect.Value {
	if st, ok := t.(scriggoType); ok {
		return Zero(st.Underlying())
	}
	return reflect.Zero(t)
}
