// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import "reflect"

// ChanOf behaves like reflect.ChanOf except when t is a Scriggo type; in such
// case a new Scriggo channel type is created and returned as reflect.Type.
func ChanOf(dir reflect.ChanDir, t reflect.Type) reflect.Type {
	if st, ok := t.(ScriggoType); ok {
		return chanType{
			Type: ChanOf(dir, st.Underlying()),
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

func (x chanType) AssignableTo(T reflect.Type) bool {
	return x == T
}

func (x chanType) Underlying() reflect.Type {
	assertNotScriggoType(x.Type)
	return x.Type
}

func (x chanType) Elem() reflect.Type {
	return x.elem
}

func (x chanType) String() string {
	var s string
	switch x.ChanDir() {
	case reflect.BothDir:
		s += "chan "
	case reflect.RecvDir:
		s += "<-chan "
	case reflect.SendDir:
		s += "chan<- "
	}
	s += x.elem.String()
	return s
}
