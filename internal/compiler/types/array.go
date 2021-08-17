// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"reflect"
	"strconv"

	"github.com/open2b/scriggo/internal/runtime"
)

// ArrayOf is equivalent to reflect.ArrayOf except when elem is a Scriggo type;
// in such case a new Scriggo array type is created and returned as
// reflect.Type.
func (types *Types) ArrayOf(count int, elem reflect.Type) reflect.Type {
	if st, ok := elem.(runtime.ScriggoType); ok {
		return arrayType{
			Type: reflect.ArrayOf(count, st.GoType()),
			elem: st,
		}
	}
	return reflect.ArrayOf(count, elem)
}

// arrayType represents a composite array type where the element is a Scriggo
// type.
type arrayType struct {
	reflect.Type              // always a reflect implementation of reflect.Type
	elem         reflect.Type // array element, always a Scriggo type
}

func (x arrayType) AssignableTo(y reflect.Type) bool {
	return AssignableTo(x, y)
}

func (x arrayType) ConvertibleTo(y reflect.Type) bool {
	return ConvertibleTo(x, y)
}

func (x arrayType) Elem() reflect.Type {
	return x.elem
}

func (x arrayType) Implements(y reflect.Type) bool {
	return Implements(x, y)
}

func (x arrayType) Name() string {
	return "" // composite types do not have a name.
}

func (x arrayType) String() string {
	return "[" + strconv.Itoa(x.Type.Len()) + "]" + x.elem.String()
}

// GoType implements the interface runtime.ScriggoType.
func (x arrayType) GoType() reflect.Type {
	assertNotScriggoType(x.Type)
	return x.Type
}

// Unwrap implements the interface runtime.ScriggoType.
func (x arrayType) Unwrap(v reflect.Value) (reflect.Value, bool) { return unwrap(x, v) }

// Wrap implements the interface runtime.ScriggoType.
func (x arrayType) Wrap(v reflect.Value) reflect.Value { return wrap(x, v) }
