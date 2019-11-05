// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import "reflect"

func ChanOf(dir reflect.ChanDir, t reflect.Type) reflect.Type {
	if st, ok := t.(ScriggoType); ok {
		return chanType{
			Type: ChanOf(dir, st.Underlying()),
			elem: st,
		}
	}
	return reflect.ChanOf(dir, t)
}

type chanType struct {
	reflect.Type
	elem reflect.Type // always != nil
}

func (x chanType) Underlying() reflect.Type {
	return x.Type
}

func (x chanType) Elem() reflect.Type {
	// x.elem is always != nil, otherwise this chanType has no reason to exist.
	return x.elem
}
