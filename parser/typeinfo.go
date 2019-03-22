// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parser

import (
	"reflect"
)

type Properties uint8

const (
	PropertyNil              Properties = 1 << (8 - 1 - iota) // is predeclared nil
	PropertyUntyped                                           // is untyped
	PropertyIsConstant                                        // is a constant
	PropertyIsType                                            // is a type
	PropertyIsPackage                                         // is a package
	PropertyIsBuiltin                                         // is a builtin
	PropertyAddressable                                       // is addressable
	PropertyMustBeReferenced                                  // must be referenced when declared
)

type TypeInfo struct {
	Type       reflect.Type // Type.
	Properties Properties   // Properties.
	Value      interface{}  // Value; for packages has type *Package.
}

// Nil reports whether it is the predeclared nil.
func (ti *TypeInfo) Nil() bool {
	return ti.Properties&PropertyNil != 0
}

// Untyped reports whether it is untyped.
func (ti *TypeInfo) Untyped() bool {
	return ti.Properties&PropertyUntyped != 0
}

// IsConstant reports whether it is a constant.
func (ti *TypeInfo) IsConstant() bool {
	return ti.Properties&PropertyIsConstant != 0
}

// IsType reports whether it is a type.
func (ti *TypeInfo) IsType() bool {
	return ti.Properties&PropertyIsType != 0
}

// IsPackage reports whether it is a package.
func (ti *TypeInfo) IsPackage() bool {
	return ti.Properties&PropertyIsPackage != 0
}

// IsBuiltin reports whether it is a builtin.
func (ti *TypeInfo) IsBuiltin() bool {
	return ti.Properties&PropertyIsBuiltin != 0
}

// Addressable reports whether it is addressable.
func (ti *TypeInfo) Addressable() bool {
	return ti.Properties&PropertyAddressable != 0
}

// MustBeReferenced reports whether ti must be referenced when gets declared.
func (ti *TypeInfo) MustBeReferenced() bool {
	return ti.Properties&PropertyMustBeReferenced != 0
}

var runeType = reflect.TypeOf(rune(0))

// String returns a string representation.
func (ti *TypeInfo) String() string {
	if ti.Nil() {
		return "nil"
	}
	var s string
	if ti.Untyped() {
		s = "untyped "
	}
	if ti.IsConstant() && ti.Type == runeType {
		return s + "rune"
	}
	return s + ti.Type.String()
}

// ShortString returns a short string representation.
func (ti *TypeInfo) ShortString() string {
	if ti.Nil() {
		return "nil"
	}
	if ti.IsConstant() && ti.Type == runeType {
		return "rune"
	}
	return ti.Type.String()
}

// StringWithNumber returns the string representation of ti in the context of a
// function call and return statement.
func (ti *TypeInfo) StringWithNumber(explicitUntyped bool) string {
	if ti.Untyped() && ti.IsNumeric() {
		if explicitUntyped {
			return "untyped number"
		}
		return "number"
	}
	return ti.Type.String()
}

// IsNumeric reports whether it is numeric.
func (ti *TypeInfo) IsNumeric() bool {
	k := ti.Type.Kind()
	return !ti.Nil() && reflect.Int <= k && k <= reflect.Complex128
}

// IsInteger reports whether it is an integer.
func (ti *TypeInfo) IsInteger() bool {
	k := ti.Type.Kind()
	return !ti.Nil() && reflect.Int <= k && k <= reflect.Uintptr
}
