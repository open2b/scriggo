// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import "reflect"

func PtrTo(t reflect.Type) reflect.Type {
	if st, ok := t.(ScriggoType); ok {
		return ptrType{
			Type: PtrTo(st.Underlying()),
			elem: st,
		}
	}
	return reflect.PtrTo(t)
}

type ptrType struct {
	reflect.Type
	elem reflect.Type // always != nil
}

func (x ptrType) Underlying() reflect.Type {
	return x.Type
}

func (x ptrType) Elem() reflect.Type {
	// x.elem is always != nil, otherwise this ptrType has no reason to exist.
	return x.elem
}
