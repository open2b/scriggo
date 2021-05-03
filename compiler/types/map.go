// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import "reflect"

// MapOf behaves like reflect.MapOf except when at least one of the map key or
// the map element is a non-native type; in such case a new non-native map
// type is created and returned as reflect.Type.
func (types *Types) MapOf(key, elem reflect.Type) reflect.Type {
	if keySt, ok := key.(Type); ok {
		if elemSt, ok := elem.(Type); ok {
			return mapType{
				Type: reflect.MapOf(keySt.Underlying(), elemSt.Underlying()),
				key:  keySt,
				elem: elemSt,
			}
		}
		return mapType{
			Type: reflect.MapOf(keySt.Underlying(), elem),
			key:  keySt,
		}
	}
	if elemSt, ok := elem.(Type); ok {
		return mapType{
			Type: reflect.MapOf(key, elemSt.Underlying()),
			elem: elemSt,
		}
	}
	return reflect.MapOf(key, elem)
}

// mapType represents a composite map type where at least one of map key or
// map element is a non-native type.
type mapType struct {
	reflect.Type
	key, elem reflect.Type
}

// AssignableTo is equivalent to reflect's AssignableTo.
func (x mapType) AssignableTo(u reflect.Type) bool {
	return x == u
}

func (x mapType) Elem() reflect.Type {
	if x.elem != nil {
		return x.elem
	}
	return x.Type.Elem()
}

func (x mapType) Implements(u reflect.Type) bool {
	if u.Kind() != reflect.Interface {
		panic("expected reflect.Interface")
	}
	return u.NumMethod() == 0
}

func (x mapType) Name() string {
	return "" // composite types do not have a name.
}

func (x mapType) Key() reflect.Type {
	if x.key != nil {
		return x.key
	}
	return x.Type.Key()
}

func (x mapType) String() string {
	s := "map["
	if x.key == nil {
		s += x.Type.Key().String()
	} else {
		s += x.key.String()
	}
	s += "]"
	if x.elem == nil {
		s += x.Type.Elem().String()
	} else {
		s += x.elem.String()
	}
	return s
}

// Underlying implements the interface runtime.Wrapper.
func (x mapType) Underlying() reflect.Type {
	assertNativeType(x.Type)
	return x.Type
}

// Unwrap implements the interface runtime.Wrapper.
func (x mapType) Unwrap(v reflect.Value) (reflect.Value, bool) { return unwrap(x, v) }

// Wrap implements the interface runtime.Wrapper.
func (x mapType) Wrap(v reflect.Value) reflect.Value { return wrap(x, v) }
