// Copyright (c) 2020 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"reflect"

	"github.com/open2b/scriggo/runtime"
)

// Runtime implements the runtime.Types interface.
type Runtime struct {
	types *Types
}

func (rt Runtime) New(typ reflect.Type) reflect.Value {
	return rt.types.New(typ)
}

func (rt Runtime) TypeOf(i interface{}) reflect.Type {
	return rt.types.TypeOf(reflect.ValueOf(i))
}

func (rt Runtime) ValueOf(i interface{}) reflect.Value {
	if i == nil {
		return reflect.Value{}
	}
	v := reflect.ValueOf(i)
	t := rt.types.TypeOf(v)
	if w, ok := t.(runtime.Wrapper); ok {
		v, _ = w.Unwrap(v)
	}
	return v
}
