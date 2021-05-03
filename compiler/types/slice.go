// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import "reflect"

// SliceOf behaves like reflect.SliceOf except when elem is a non-native type;
// in such case a new non-native slice type is created and returned as
// reflect.Type.
func (types *Types) SliceOf(t reflect.Type) reflect.Type {
	if st, ok := t.(Type); ok {
		return sliceType{
			Type: reflect.SliceOf(st.Underlying()),
			elem: st,
		}
	}
	return reflect.SliceOf(t)
}

// sliceType represents a composite slice type where the element is a
// non-native type.
type sliceType struct {
	reflect.Type
	elem reflect.Type
}

// AssignableTo is equivalent to reflect's AssignableTo.
func (x sliceType) AssignableTo(u reflect.Type) bool {
	return x == u
}

func (x sliceType) Elem() reflect.Type {
	return x.elem
}

func (x sliceType) Implements(u reflect.Type) bool {
	if u.Kind() != reflect.Interface {
		panic("expected reflect.Interface")
	}
	return u.NumMethod() == 0
}

func (x sliceType) Name() string {
	return "" // composite types do not have a name.
}

func (x sliceType) String() string {
	return "[]" + (x.elem).String()
}

// Underlying implements the interface runtime.Wrapper.
func (x sliceType) Underlying() reflect.Type {
	assertNativeType(x.Type)
	return x.Type
}

// Unwrap implements the interface runtime.Wrapper.
func (x sliceType) Unwrap(v reflect.Value) (reflect.Value, bool) { return unwrap(x, v) }

// Wrap implements the interface runtime.Wrapper.
func (x sliceType) Wrap(v reflect.Value) reflect.Value { return wrap(x, v) }
