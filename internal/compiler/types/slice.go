// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import "reflect"

func SliceOf(t reflect.Type) reflect.Type {
	if st, ok := t.(ScriggoType); ok {
		return sliceType{
			Type: SliceOf(st.Underlying()),
			elem: st,
		}
	}
	return reflect.SliceOf(t)
}

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
