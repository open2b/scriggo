// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"reflect"
	"strconv"
)

func ArrayOf(count int, elem reflect.Type) reflect.Type {
	if st, ok := elem.(ScriggoType); ok {
		return arrayType{
			Type: ArrayOf(count, st.Underlying()),
			elem: st,
		}
	}
	return reflect.ArrayOf(count, elem)
}

type arrayType struct {
	reflect.Type
	elem reflect.Type
}

func (x arrayType) AssignableTo(T reflect.Type) bool {
	return x == T
}

func (x arrayType) Name() string {
	return ""
}

func (x arrayType) Elem() reflect.Type {
	return x.elem
}

func (x arrayType) Underlying() reflect.Type {
	return x.Type
}

func (x arrayType) String() string {
	s := "[" + strconv.Itoa(x.Type.Len()) + "]"
	s += x.elem.Name()
	return s
}
