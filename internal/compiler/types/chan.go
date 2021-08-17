// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"reflect"

	"github.com/open2b/scriggo/internal/runtime"
)

// ChanOf behaves like reflect.ChanOf except when t is a Scriggo type; in such
// case a new Scriggo channel type is created and returned as reflect.Type.
func (types *Types) ChanOf(dir reflect.ChanDir, t reflect.Type) reflect.Type {
	if st, ok := t.(runtime.ScriggoType); ok {
		return chanType{
			Type: reflect.ChanOf(dir, st.GoType()),
			elem: st,
		}
	}
	return reflect.ChanOf(dir, t)
}

// chanType represents a composite channel type where the element is a Scriggo
// type.
type chanType struct {
	reflect.Type              // always a reflect implementation of reflect.Type
	elem         reflect.Type // channel element, always a Scriggo type
}

func (x chanType) AssignableTo(y reflect.Type) bool {
	return AssignableTo(x, y)
}

func (x chanType) ConvertibleTo(y reflect.Type) bool {
	return ConvertibleTo(x, y)
}

func (x chanType) Elem() reflect.Type {
	return x.elem
}

func (x chanType) Implements(y reflect.Type) bool {
	return Implements(x, y)
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

// GoType implements the interface runtime.ScriggoType.
func (x chanType) GoType() reflect.Type {
	assertNotScriggoType(x.Type)
	return x.Type
}

// Unwrap implements the interface runtime.ScriggoType.
func (x chanType) Unwrap(v reflect.Value) (reflect.Value, bool) { return unwrap(x, v) }

// Wrap implements the interface runtime.ScriggoType.
func (x chanType) Wrap(v reflect.Value) reflect.Value { return wrap(x, v) }
