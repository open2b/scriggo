// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"reflect"
)

// newScriggoType creates a new type defined in Scriggo with the syntax
//
//     type Int int
//
func newScriggoType(name string, baseType reflect.Type) scriggoType {
	return scriggoType{
		Type: baseType,
		name: name,
	}
}

type scriggoType struct {
	reflect.Type
	name string
	// Path string
	// Methods []Method
}

func (st scriggoType) AssignableTo(T reflect.Type) bool {
	if T, ok := T.(scriggoType); ok {
		return st.name == T.name
	}
	panic("not implemented")
}

func (st scriggoType) String() string {
	return st.name // TODO
}
