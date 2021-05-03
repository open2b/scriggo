// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import "reflect"

// ChanOf behaves like reflect.ChanOf except when t is a non-native type; in
// such case a new non-native channel type is created and returned as
// reflect.Type.
func (types *Types) ChanOf(dir reflect.ChanDir, t reflect.Type) reflect.Type {
	if st, ok := t.(Type); ok {
		return chanType{
			Type: reflect.ChanOf(dir, st.Underlying()),
			elem: st,
		}
	}
	return reflect.ChanOf(dir, t)
}

// chanType represents a composite channel type where the element is a
// non-native type.
type chanType struct {
	reflect.Type              // always a reflect implementation of reflect.Type
	elem         reflect.Type // channel element, always a non-native type
}

// AssignableTo is equivalent to reflect's AssignableTo.
func (x chanType) AssignableTo(u reflect.Type) bool {
	return x == u
}

func (x chanType) Elem() reflect.Type {
	return x.elem
}

func (x chanType) Implements(u reflect.Type) bool {
	if u.Kind() != reflect.Interface {
		panic("expected reflect.Interface")
	}
	return u.NumMethod() == 0
}

func (x chanType) Name() string {
	return "" // composite types do not have a name.
}

func (x chanType) String() string {
	var s string
	switch x.ChanDir() {
	case reflect.BothDir:
		s = "chan "
	case reflect.RecvDir:
		s = "<-chan "
	case reflect.SendDir:
		s = "chan<- "
	}
	s += x.elem.String()
	return s
}

// Underlying implements the interface runtime.Wrapper.
func (x chanType) Underlying() reflect.Type {
	assertNativeType(x.Type)
	return x.Type
}

// Unwrap implements the interface runtime.Wrapper.
func (x chanType) Unwrap(v reflect.Value) (reflect.Value, bool) { return unwrap(x, v) }

// Wrap implements the interface runtime.Wrapper.
func (x chanType) Wrap(v reflect.Value) reflect.Value { return wrap(x, v) }
