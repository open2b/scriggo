// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import "reflect"

// SliceOf behaves like reflect.SliceOf except when elem is a Scriggo type; in
// such case a new Scriggo slice type is created and returned as reflect.Type.
func (types *Types) SliceOf(t reflect.Type) reflect.Type {
	if st, ok := t.(ScriggoType); ok {
		return sliceType{
			Type: reflect.SliceOf(st.Underlying()),
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

func (x sliceType) AssignableTo(T reflect.Type) bool {
	return x == T
}

func (x sliceType) Name() string {
	return ""
}

func (x sliceType) Implements(u reflect.Type) bool {
	if u.Kind() != reflect.Interface {
		panic("expected reflect.Interface")
	}
	return u.NumMethod() == 0
}

func (x sliceType) Elem() reflect.Type {
	return x.elem
}

func (x sliceType) Underlying() reflect.Type {
	assertNotScriggoType(x.Type)
	return x.Type
}

func (x sliceType) String() string {
	return "[]" + (x.elem).String()
}

func (x sliceType) Wrap(v reflect.Value) reflect.Value { return wrap(x, v) }

func (x sliceType) Unwrap(v reflect.Value) (reflect.Value, bool) { return unwrap(x, v) }
