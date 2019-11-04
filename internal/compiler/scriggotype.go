// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import "reflect"

type scriggoType struct {
	baseType reflect.Type
	name     string
	// Path string
	// Methods []Method
}

func newScriggoType(name string, baseType reflect.Type) scriggoType {
	return scriggoType{
		baseType: baseType,
		name:     name,
	}
}
