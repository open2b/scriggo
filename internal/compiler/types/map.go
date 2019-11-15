// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import "reflect"

// MapOf behaves like reflect.MapOf except when at least once of the map key or
// the map element is a Scriggo type; in such case a new Scriggo map type is
// created and returned as reflect.Type.
func (types *Types) MapOf(key, elem reflect.Type) reflect.Type {
	keySt, keyIsScriggoType := key.(ScriggoType)
	elemSt, elemIsScriggoType := elem.(ScriggoType)
	switch {
	case keyIsScriggoType && elemIsScriggoType:
		return mapType{
			Type: reflect.MapOf(keySt.Underlying(), elemSt.Underlying()),
			key:  keySt,
			elem: elemSt,
		}
	case keyIsScriggoType && !elemIsScriggoType:
		return mapType{
			Type: reflect.MapOf(keySt.Underlying(), elem),
			key:  keySt,
		}
	case elemIsScriggoType && !keyIsScriggoType:
		return mapType{
			Type: reflect.MapOf(key, elemSt.Underlying()),
			elem: elemSt,
		}
	default:
		return reflect.MapOf(key, elem)
	}
}

// mapType represents a composite map type where at least once of map key or map
// element is a Scriggo type.
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
	assertNotScriggoType(x.Type)
	return x.Type
}

// Unwrap implement the interface runtime.Wrapper.
func (x mapType) Unwrap(v reflect.Value) (reflect.Value, bool) { return unwrap(x, v) }

// Wrap implement the interface runtime.Wrapper.
func (x mapType) Wrap(v reflect.Value) reflect.Value { return wrap(x, v) }
