// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"reflect"
	"strconv"
)

// ArrayOf behaves like reflect.ArrayOf except when elem is a Scriggo type; in
// such case a new Scriggo array type is created and returned as reflect.Type.
func (types *Types) ArrayOf(count int, elem reflect.Type) reflect.Type {
	if st, ok := elem.(ScriggoType); ok {
		return arrayType{
			Type: types.ArrayOf(count, st.Underlying()),
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

func (x arrayType) AssignableTo(T reflect.Type) bool {
	return x == T
}

func (x arrayType) Name() string {
	return ""
}

func (x arrayType) Elem() reflect.Type {
	return x.elem
}

func (x arrayType) Underlying() reflect.Type {
	assertNotScriggoType(x.Type)
	return x.Type
}

func (x arrayType) String() string {
	s := "[" + strconv.Itoa(x.Type.Len()) + "]"
	s += x.elem.String()
	return s
}

func (x arrayType) Wrap(v interface{}) interface{} { return wrap(x, v) }

func (x arrayType) Unwrap(v reflect.Value) (reflect.Value, bool) { return unwrap(x, v) }
