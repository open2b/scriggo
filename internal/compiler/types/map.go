// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"reflect"

	"github.com/open2b/scriggo/internal/runtime"
)

// MapOf behaves like reflect.MapOf except when at least one of the map key or
// the map element is a Scriggo type; in such case a new Scriggo map type is
// created and returned as reflect.Type.
func (types *Types) MapOf(key, elem reflect.Type) reflect.Type {
	keySt, keyIsScriggoType := key.(runtime.ScriggoType)
	elemSt, elemIsScriggoType := elem.(runtime.ScriggoType)
	switch {
	case keyIsScriggoType && elemIsScriggoType:
		return mapType{
			Type: reflect.MapOf(keySt.GoType(), elemSt.GoType()),
			key:  keySt,
			elem: elemSt,
		}
	case keyIsScriggoType && !elemIsScriggoType:
		return mapType{
			Type: reflect.MapOf(keySt.GoType(), elem),
			key:  keySt,
		}
	case elemIsScriggoType && !keyIsScriggoType:
		return mapType{
			Type: reflect.MapOf(key, elemSt.GoType()),
			elem: elemSt,
		}
	default:
		return reflect.MapOf(key, elem)
	}
}

// mapType represents a composite map type where at least one of map key or
// map element is a Scriggo type.
type mapType struct {
	reflect.Type
	key, elem reflect.Type
}

func (x mapType) AssignableTo(y reflect.Type) bool {
	return AssignableTo(x, y)
}

func (x mapType) ConvertibleTo(y reflect.Type) bool {
	return ConvertibleTo(x, y)
}

func (x mapType) Elem() reflect.Type {
	if x.elem != nil {
		return x.elem
	}
	return x.Type.Elem()
}

func (x mapType) Implements(y reflect.Type) bool {
	return Implements(x, y)
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

// GoType implements the interface runtime.ScriggoType.
func (x mapType) GoType() reflect.Type {
	assertNotScriggoType(x.Type)
	return x.Type
}

// Unwrap implements the interface runtime.ScriggoType.
func (x mapType) Unwrap(v reflect.Value) (reflect.Value, bool) { return unwrap(x, v) }

// Wrap implements the interface runtime.ScriggoType.
func (x mapType) Wrap(v reflect.Value) reflect.Value { return wrap(x, v) }
