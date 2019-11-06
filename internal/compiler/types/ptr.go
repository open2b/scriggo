// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import "reflect"

// PtrTo behaves like reflect.PtrTo except when t is a Scriggo type; in such
// case a new Scriggo pointer type is created and returned as reflect.Type.
func (types *Types) PtrTo(t reflect.Type) reflect.Type {
	if st, ok := t.(ScriggoType); ok {
		return ptrType{
			Type: types.PtrTo(st.Underlying()),
			elem: st,
		}
	}
	return reflect.PtrTo(t)
}

// ptrType represents a composite pointer type where the element is a Scriggo
// type.
type ptrType struct {
	reflect.Type
	elem reflect.Type // always != nil
}

func (x ptrType) AssignableTo(T reflect.Type) bool {
	return x == T
}

func (x ptrType) Underlying() reflect.Type {
	assertNotScriggoType(x.Type)
	return x.Type
}

func (x ptrType) Elem() reflect.Type {
	// x.elem is always != nil, otherwise this ptrType has no reason to exist.
	return x.elem
}

func (x ptrType) String() string {
	return "*" + x.elem.String()
}

func (x ptrType) Wrap(v interface{}) interface{} { return wrap(x, v) }

func (x ptrType) Unwrap(v reflect.Value) (reflect.Value, bool) { return unwrap(x, v) }
