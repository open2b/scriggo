// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import "reflect"

// wrap and unwrap are called by the methods Wrap and Unwrap of the types
// defined in this package. These two methods and the Underlying method
// implement the runtime.Wrapper interface.

func wrap(t Type, v reflect.Value) reflect.Value {
	return reflect.ValueOf(emptyInterfaceProxy{
		value: v,
		sign:  t,
	})
}

// TODO: currently unwrap always returns an empty interface wrapper. This will
//  change when methods declaration will be implemented in Scriggo.
func unwrap(x Type, v reflect.Value) (reflect.Value, bool) {
	p, ok := v.Interface().(emptyInterfaceProxy)
	// Not a proxy.
	if !ok {
		return reflect.Value{}, false
	}
	// v is a proxy but it has a different non-native type.
	if p.sign != x {
		return reflect.Value{}, false
	}
	return p.value, true
}

// emptyInterfaceProxy is a proxy for values of types that have an empty
// method set.
type emptyInterfaceProxy struct {
	value reflect.Value
	sign  Type
}
