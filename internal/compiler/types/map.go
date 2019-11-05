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
		return scriggoMapType{
			Type: MapOf(keySt.Underlying(), elemSt.Underlying()),
			key:  keySt,
			elem: elemSt,
		}
	case keyIsScriggoType && !elemIsScriggoType:
		return scriggoMapType{
			Type: MapOf(keySt.Underlying(), elem),
			key:  keySt,
		}
	case elemIsScriggoType && !keyIsScriggoType:
		return scriggoMapType{
			Type: MapOf(key, elemSt.Underlying()),
			elem: elemSt,
		}
	default:
		return reflect.MapOf(key, elem)
	}
}

type scriggoMapType struct {
	reflect.Type
	key, elem reflect.Type
}

func (x scriggoMapType) Elem() reflect.Type {
	if x.elem != nil {
		return x.elem
	}
	return x.Type.Elem()
}

func (x scriggoMapType) Key() reflect.Type {
	if x.key != nil {
		return x.key
	}
	return x.Type.Key()
}
