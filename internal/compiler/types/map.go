// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import "reflect"

func MapOf(key, elem reflect.Type) reflect.Type {
	keySt, keyIsScriggoType := key.(ScriggoType)
	elemSt, elemIsScriggoType := elem.(ScriggoType)
	switch {
	case keyIsScriggoType && elemIsScriggoType:
		return mapType{
			Type: MapOf(keySt.Underlying(), elemSt.Underlying()),
			key:  keySt,
			elem: elemSt,
		}
	case keyIsScriggoType && !elemIsScriggoType:
		return mapType{
			Type: MapOf(keySt.Underlying(), elem),
			key:  keySt,
		}
	case elemIsScriggoType && !keyIsScriggoType:
		return mapType{
			Type: MapOf(key, elemSt.Underlying()),
			elem: elemSt,
		}
	default:
		return reflect.MapOf(key, elem)
	}
}

type mapType struct {
	reflect.Type
	key, elem reflect.Type
}

func (x mapType) AssignableTo(T reflect.Type) bool {
	return x == T
}

func (x mapType) Elem() reflect.Type {
	if x.elem != nil {
		return x.elem
	}
	return x.Type.Elem()
}

func (x mapType) Key() reflect.Type {
	if x.key != nil {
		return x.key
	}
	return x.Type.Key()
}

func (x mapType) String() string {
	s := "map["
	if x.key != nil {
		s += x.key.String()
	} else {
		s += x.Type.Key().String()
	}
	s += "]"
	if x.elem != nil {
		s += x.elem.String()
	} else {
		s += x.Type.Elem().String()
	}
	return s
}

func (x mapType) Underlying() reflect.Type {
	assertNotScriggoType(x.Type)
	return x.Type
}
