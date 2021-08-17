// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"reflect"

	"github.com/open2b/scriggo/internal/runtime"
)

// SliceOf behaves like reflect.SliceOf except when elem is a Scriggo type; in
// such case a new Scriggo slice type is created and returned as reflect.Type.
func (types *Types) SliceOf(t reflect.Type) reflect.Type {
	if st, ok := t.(runtime.ScriggoType); ok {
		return sliceType{
			Type: reflect.SliceOf(st.GoType()),
			elem: st,
		}
	}
	return reflect.SliceOf(t)
}

// sliceType represents a composite slice type where the element is a Scriggo
// type.
type sliceType struct {
	reflect.Type
	elem reflect.Type
}

func (x sliceType) AssignableTo(y reflect.Type) bool {
	return AssignableTo(x, y)
}

func (x sliceType) ConvertibleTo(y reflect.Type) bool {
	return ConvertibleTo(x, y)
}

func (x sliceType) Elem() reflect.Type {
	return x.elem
}

func (x sliceType) Implements(y reflect.Type) bool {
	return Implements(x, y)
}

func (x sliceType) Name() string {
	return "" // composite types do not have a name.
}

func (x sliceType) String() string {
	return "[]" + (x.elem).String()
}

// GoType implements the interface runtime.ScriggoType.
func (x sliceType) GoType() reflect.Type {
	assertNotScriggoType(x.Type)
	return x.Type
}

// Unwrap implements the interface runtime.ScriggoType.
func (x sliceType) Unwrap(v reflect.Value) (reflect.Value, bool) { return unwrap(x, v) }

// Wrap implements the interface runtime.ScriggoType.
func (x sliceType) Wrap(v reflect.Value) reflect.Value { return wrap(x, v) }
