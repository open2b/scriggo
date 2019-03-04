// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ast

import (
	"go/constant"
	"reflect"
)

type DefaultType uint8

const (
	DefaultTypeInt     DefaultType = iota // int
	DefaultTypeRune                       // rune
	DefaultTypeFloat64                    // float64
	DefaultTypeComplex                    // complex
	DefaultTypeString                     // string
	DefaultTypeBool                       // bool

)

var defaultTypeString = [...]string{
	DefaultTypeInt:     "int",
	DefaultTypeRune:    "rune",
	DefaultTypeFloat64: "float64",
	DefaultTypeComplex: "complex",
	DefaultTypeString:  "string",
	DefaultTypeBool:    "bool",
}

func (dt DefaultType) String() string {
	return defaultTypeString[dt]
}

type Constant struct {
	DefaultType DefaultType
	Bool        bool
	String      string
	Number      constant.Value
}

type Properties uint8

const (
	PropertyNil         Properties = 1 << (8 - 1 - iota) // is predeclared nil
	PropertyIsType                                       // is a type
	PropertyIsPackage                                    // is a package
	PropertyAddressable                                  // is addressable
)

type TypeInfo struct {
	Type       reflect.Type // Type; nil if untyped.
	Properties Properties   // Properties.
	Constant   *Constant    // Constant value; nil if not constant.
	Package    *Package     // Package; nil if not a package.
}

// Nil reports whether it is the predeclared nil.
func (ti *TypeInfo) Nil() bool {
	return ti.Properties&PropertyNil != 0
}

// IsType reports whether it is a type.
func (ti *TypeInfo) IsType() bool {
	return ti.Properties&PropertyIsType != 0
}

// IsPackage reports whether it is a package.
func (ti *TypeInfo) IsPackage() bool {
	return ti.Properties&PropertyIsPackage != 0
}

// Addressable reports whether it is a addressable.
func (ti *TypeInfo) Addressable() bool {
	return ti.Properties&PropertyAddressable != 0
}

// String returns a string representation.
func (ti *TypeInfo) String() string {
	if ti.Nil() {
		return "nil"
	}
	var s string
	if ti.Type == nil {
		s = "untyped "
	}
	if ti.Constant != nil {
		return s + ti.Constant.DefaultType.String()
	}
	if ti.Type == nil {
		return s + "bool"
	}
	return ti.Type.String()
}

// ShortString returns a short string representation.
func (ti *TypeInfo) ShortString() string {
	switch {
	case ti.Nil():
		return "nil"
	case ti.Constant != nil:
		return ti.Constant.DefaultType.String()
	case ti.Type == nil:
		return "bool"
	default:
		return ti.Type.String()
	}
}
