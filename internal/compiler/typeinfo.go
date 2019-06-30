// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"reflect"
)

const minIntAsFloat64 = 1 << 53

type Properties uint8

const (
	PropertyNil          Properties = 1 << (8 - 1 - iota) // is predeclared nil
	PropertyUntyped                                       // is untyped
	PropertyIsType                                        // is a type
	PropertyIsPackage                                     // is a package
	PropertyIsBuiltin                                     // is a builtin
	PropertyAddressable                                   // is addressable
	PropertyIsPredefined                                  // is predefined
)

type TypeInfo struct {
	Type              reflect.Type // Type.
	Properties        Properties   // Properties.
	Constant          constant     // Constant value.
	Value             interface{}  // Value; for packages has type *Package.
	PredefPackageName string       // Name of the package. Empty string if not predefined.
	MethodType        MethodType   // Method type.
}

// MethodType represents the type of a method, intended as a combination of a
// method call/value/expression and a receiver type (concrete or interface).
type MethodType uint8

const (
	NoMethod             MethodType = iota // Not a method.
	MethodValueConcrete                    // Method value on a concrete receiver.
	MethodValueInterface                   // Method value on an interface receiver.
	MethodCallConcrete                     // Method call on concrete receiver.
	MethodCallInterface                    // Method call on interface receiver.
)

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
	return ti.Constant != nil
}

// IsUntypedConstant reports whether it is an untyped constant.
func (ti *TypeInfo) IsUntypedConstant() bool {
	return ti.Properties&PropertyUntyped != 0 && ti.Constant != nil
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

// IsPredefined reports whether it is predefined.
func (ti *TypeInfo) IsPredefined() bool {
	return ti.Properties&PropertyIsPredefined != 0
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
	if ti.Nil() {
		return false
	}
	k := ti.Type.Kind()
	return reflect.Int <= k && k <= reflect.Complex128
}

// IsInteger reports whether it is an integer.
func (ti *TypeInfo) IsInteger() bool {
	if ti.Nil() {
		return false
	}
	k := ti.Type.Kind()
	return reflect.Int <= k && k <= reflect.Uintptr
}

// IsUnsignedInteger reports whether it is an unsigned integer.
func (ti *TypeInfo) IsUnsignedInteger() bool {
	if ti.Nil() {
		return false
	}
	k := ti.Type.Kind()
	return reflect.Uint <= k && k <= reflect.Uint64
}
