// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import "reflect"

// PtrTo behaves like reflect.PtrTo except when it is a non-native type; in
// such case a new non-native pointer type is created and returned as
// reflect.Type.
func (types *Types) PtrTo(t reflect.Type) reflect.Type {
	if st, ok := t.(Type); ok {
		return ptrType{
			Type: reflect.PtrTo(st.Underlying()),
			elem: st,
		}
	}
	return reflect.PtrTo(t)
}

// ptrType represents a composite pointer type where the element is a
// non-native type.
type ptrType struct {
	reflect.Type
	elem reflect.Type // Cannot be nil.
}

// AssignableTo is equivalent to reflect's AssignableTo.
func (x ptrType) AssignableTo(u reflect.Type) bool {
	return x == u
}

func (x ptrType) Elem() reflect.Type {
	return x.elem
}

func (x ptrType) Implements(u reflect.Type) bool {
	if u.Kind() != reflect.Interface {
		panic("expected reflect.Interface")
	}
	return u.NumMethod() == 0
}

func (x ptrType) Name() string {
	return "" // composite types do not have a name.
}

func (x ptrType) String() string {
	return "*" + x.elem.String()
}

// Underlying implements the interface runtime.Wrapper.
func (x ptrType) Underlying() reflect.Type {
	assertNativeType(x.Type)
	return x.Type
}

// Unwrap implements the interface runtime.Wrapper.
func (x ptrType) Unwrap(v reflect.Value) (reflect.Value, bool) { return unwrap(x, v) }

// Wrap implements the interface runtime.Wrapper.
func (x ptrType) Wrap(v reflect.Value) reflect.Value { return wrap(x, v) }
