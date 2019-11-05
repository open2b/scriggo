// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import "reflect"

func SliceOf(t reflect.Type) reflect.Type {
	if st, ok := t.(ScriggoType); ok {
		return scriggoSliceType{
			Type: SliceOf(st.Underlying()),
			elem: st,
		}
	}
	return reflect.SliceOf(t)
}

type scriggoSliceType struct {
	reflect.Type
	elem reflect.Type
}

func (x scriggoSliceType) AssignableTo(T reflect.Type) bool {
	return x == T
}

func (x scriggoSliceType) Name() string {
	return ""
}

func (x scriggoSliceType) Elem() reflect.Type {
	return x.elem
}

func (x scriggoSliceType) Underlying() reflect.Type {
	return x.Type
}

func (x scriggoSliceType) String() string {
	return "[]" + (x.elem).String()
}
